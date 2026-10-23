import ast
import json
import sys

# Fixed scanner code: source arrives only as data; never import the target.
tree = ast.parse(sys.stdin.buffer.read())
MAX = 256
functions = []
classes = {}
parents = {}
for node in ast.walk(tree):
    for child in ast.iter_child_nodes(node):
        parents[child] = node

def mutable(node):
    return isinstance(node, (ast.Dict, ast.List, ast.Set)) or (
        isinstance(node, ast.Call) and isinstance(node.func, ast.Name)
        and node.func.id in ('dict', 'list', 'set'))

def assigned(scope):
    result = set()
    for node in scope.body:
        if isinstance(node, ast.Assign) and mutable(node.value):
            result.update(t.id for t in node.targets if isinstance(t, ast.Name))
        if isinstance(node, ast.AnnAssign) and mutable(node.value) and isinstance(node.target, ast.Name):
            result.add(node.target.id)
    return result

module_state = assigned(tree)
module_dicts = {t.id for n in tree.body if isinstance(n, ast.Assign) and (isinstance(n.value,ast.Dict) or isinstance(n.value,ast.Call) and isinstance(n.value.func,ast.Name) and n.value.func.id == "dict") for t in n.targets if isinstance(t,ast.Name)}
mutators = {'append', 'extend', 'insert', 'pop', 'clear', 'update', 'setdefault', 'add', 'remove', 'discard'}
cache_names = set()
functools_names = set()
for node in tree.body:
    if isinstance(node, ast.Import):
        functools_names.update(a.asname or a.name for a in node.names if a.name == 'functools')
    if isinstance(node, ast.ImportFrom) and node.module == 'functools':
        cache_names.update(a.asname or a.name for a in node.names if a.name in ('cache', 'lru_cache'))
    if isinstance(node, ast.ClassDef):
        classes[node.name] = assigned(node)

def local_nodes(fn):
    # Do not attribute a nested function's syntax to its containing function.
    stack = list(reversed(fn.body))
    while stack:
        node = stack.pop()
        yield node
        if not isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef, ast.Lambda)):
            stack.extend(reversed(list(ast.iter_child_nodes(node))))

def owner(fn):
    chain = []
    p = parents.get(fn)
    nested = False
    while p is not None and p is not tree:
        if isinstance(p, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            chain.append(p.name)
            if isinstance(p, (ast.FunctionDef, ast.AsyncFunctionDef)):
                nested = True
        p = parents.get(p)
    return '.'.join(reversed(chain + [])), nested

for node in ast.walk(tree):
    if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
        functions.append(node)
functions.sort(key=lambda n: (n.lineno, n.name))
by_name = {f.name: f for f in tree.body if isinstance(f, (ast.FunctionDef, ast.AsyncFunctionDef))}
for cls in tree.body:
    if isinstance(cls, ast.ClassDef):
        for f in cls.body:
            if isinstance(f, (ast.FunctionDef, ast.AsyncFunctionDef)):
                by_name[cls.name + '.' + f.name] = f

facts = {f: {} for f in functions}
def fact(fn, kind, node, text, tier='observation'):
    if kind not in facts[fn]:
        facts[fn][kind] = {'kind': kind, 'line': node.lineno, 'fact': text, 'tier': tier}

for fn in functions:
    prefix, nested = owner(fn)
    nodes = list(local_nodes(fn))
    args = {a.arg for a in fn.args.posonlyargs + fn.args.args + fn.args.kwonlyargs}
    # Locally rebound names are not resolved to module state.
    shadows = args | {n.id for n in nodes if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Store)}
    globals_declared = {name for n in nodes if isinstance(n, ast.Global) for name in n.names}
    shadows -= globals_declared
    state = module_state - shadows
    for n in nodes:
        if isinstance(n, ast.Name) and n.id in state:
            fact(fn, 'module_mutable_state', n, 'direct reference to module mutable name ' + n.id)
        if isinstance(n, ast.Subscript) and isinstance(n.ctx, ast.Store) and isinstance(n.value, ast.Name) and n.value.id in state:
            fact(fn, 'shared_access', n, 'direct subscript write to module mutable name ' + n.value.id)
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute):
            base = n.func.value
            if n.func.attr in mutators and isinstance(base, ast.Name) and base.id in state:
                fact(fn, 'shared_access', n, 'direct mutator ' + n.func.attr + ' on module mutable name ' + base.id)
            if (n.func.attr in mutators and isinstance(base, ast.Attribute) and isinstance(base.value, ast.Name)
                and base.value.id in ('self', 'cls') and base.attr in classes.get(prefix, set())):
                fact(fn, 'shared_access', n, 'direct mutator on class-declared mutable attribute ' + base.attr)
        if isinstance(n, ast.Subscript) and isinstance(n.value, ast.Name) and n.value.id in state & module_dicts:
            key = {v.id for v in ast.walk(n.slice) if isinstance(v, ast.Name)} & args
            used = {v.id for v in nodes if isinstance(v, ast.Name) and isinstance(v.ctx, ast.Load)} & args
            if key and key < used:
                fact(fn, 'partial_dict_key', n, 'key uses parameters ' + ','.join(sorted(key)) + '; body reads ' + ','.join(sorted(used)) + '; dependency is uncertain', 'association')
    for d in fn.decorator_list:
        call = d.func if isinstance(d, ast.Call) else d
        if (isinstance(call, ast.Name) and call.id in cache_names or
            isinstance(call, ast.Attribute) and isinstance(call.value, ast.Name) and call.value.id in functools_names and call.attr in ('cache', 'lru_cache')):
            fact(fn, 'cache_decorators', d, 'syntactically resolved functools cache decorator')
    for default in fn.args.defaults + [d for d in fn.args.kw_defaults if d is not None]:
        if mutable(default):
            fact(fn, 'mutable_defaults', default, 'mutable literal or built-in constructor in default argument')

# Calls inside loops resolve only direct module functions / class-name methods.
dynamic = []
for loop in ast.walk(tree):
    if isinstance(loop, (ast.For, ast.While, ast.ListComp, ast.SetComp, ast.DictComp, ast.GeneratorExp)):
        for call in ast.walk(loop):
            if not isinstance(call, ast.Call):
                continue
            expr = call.func
            name = expr.id if isinstance(expr, ast.Name) else (
                expr.value.id + '.' + expr.attr if isinstance(expr, ast.Attribute) and isinstance(expr.value, ast.Name) else '')
            if name in by_name:
                fact(by_name[name], 'loop_call_site', call, 'direct call ' + name + ' lexically inside loop/comprehension')
            elif isinstance(expr, ast.Attribute) and name:
                dynamic.append({'function':name, 'line':call.lineno, 'unsupported_reason':'dynamic method receiver unresolved', 'details':[{'kind':'loop_call_site','line':call.lineno,'fact':'attribute call lexically inside loop; target unresolved','tier':'observation'}]})

result = []
for fn in functions:
    prefix, nested = owner(fn)
    qualified = (prefix + '.' if prefix else '') + fn.name
    reason = ''
    if isinstance(fn, ast.AsyncFunctionDef):
        reason = 'async function unsupported'
    elif any(isinstance(n, (ast.Yield, ast.YieldFrom)) for n in local_nodes(fn)):
        reason = 'generator unsupported'
    elif nested or prefix.count('.'):
        reason = 'nested function/class unsupported'
    elif fn.name.startswith('__') and fn.name.endswith('__'):
        reason = 'dunder callable unsupported'
    elif prefix:
        for d in fn.decorator_list:
            if not isinstance(d, ast.Name) or d.id not in ('staticmethod', 'classmethod'):
                reason = 'descriptor or decorated method unsupported'
    details = list(facts[fn].values())
    if details or reason:
        result.append({'function': qualified, 'line': fn.lineno, 'unsupported_reason': reason, 'details': details})
result.extend({(d['line'],d['function']):d for d in dynamic}.values())
result.sort(key=lambda d:(d['line'],d['function']))
print(json.dumps({'items': result[:MAX], 'truncated': max(0, len(result)-MAX)}, sort_keys=True))
