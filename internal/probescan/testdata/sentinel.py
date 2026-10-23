# Synthetic: this MUST NOT execute.
raise RuntimeError("side effect sentinel executed")
async def asynchronous(): return 1
def generator(): yield 1
