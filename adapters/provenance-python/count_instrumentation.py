"""Count target edits and config lines; never count original application logic."""
import argparse
import ast
import hashlib
import json
from pathlib import Path


def count(source, baseline, config):
    text=source.read_text(encoding="utf-8-sig")
    original=baseline.read_text(encoding="utf-8-sig")
    ranges=set()
    for node in ast.walk(ast.parse(text)):
        if isinstance(node,(ast.Import,ast.ImportFrom)):
            names=[n.name for n in node.names] if isinstance(node,ast.Import) else [node.module or ""]
            if any(n.startswith(("signalscout","recorder","scout_probe")) for n in names):
                ranges.update(range(node.lineno,node.end_lineno+1))
        if isinstance(node,ast.Call):
            name=node.func.id if isinstance(node.func,ast.Name) else getattr(node.func,"attr","")
            if name in ("record_decision","Recorder"):
                ranges.update(range(node.lineno,node.end_lineno+1))
        if isinstance(node,(ast.With,ast.AsyncWith)):
            for item in node.items:
                call=item.context_expr
                if isinstance(call,ast.Call) and getattr(call.func,"attr","")=="run":
                    ranges.update(range(node.lineno,call.end_lineno+1))
    import difflib
    changed=0
    for op,a,b,c,d in difflib.SequenceMatcher(None,original.splitlines(),text.splitlines()).get_opcodes():
        if op!="equal":
            changed+=max(b-a,d-c)
    lines=config.read_text(encoding="utf-8-sig").splitlines()
    return {"source":source.name,"source_sha256":hashlib.sha256(source.read_bytes()).hexdigest(),
            "source_lines_changed":changed,"instrumentation_lines":len(ranges),
            "config":config.name,"config_physical_lines":len(lines),
            "config_nonblank_noncomment_lines":sum(bool(v.strip()) and not v.lstrip().startswith("#") for v in lines),
            "method":"Compare untouched pre-run target snapshot to post-run source; AST union of recorder imports/calls and run-context header spans only, excluding original application bodies."}


if __name__=="__main__":
    parser=argparse.ArgumentParser()
    parser.add_argument("--source",type=Path,required=True)
    parser.add_argument("--baseline",type=Path,required=True)
    parser.add_argument("--config",type=Path,required=True)
    parser.add_argument("--out",type=Path,required=True)
    args=parser.parse_args()
    args.out.write_text(json.dumps(count(args.source,args.baseline,args.config),indent=2)+"\n",encoding="utf-8")
