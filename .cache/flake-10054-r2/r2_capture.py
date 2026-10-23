# Synthetic scalar-only observation facade; original test source is unchanged.
import sys,time,json,urllib.request,urllib.error,urllib.parse
from pathlib import Path
from types import SimpleNamespace
REPO=Path(__file__).resolve().parents[2]
ADAPTER=REPO/'adapters/provenance-python'
sys.path.insert(0,str(ADAPTER));sys.path.insert(0,str(ADAPTER/'tests'))
import test_privacy
from spike_support import Serving
ORIGINAL=urllib.request.urlopen
DATA={}
def observe(kind,port,body_bytes,status,reset,elapsed_ms):
    return None

def test_request(url,data=None,timeout=5,**kwargs):
    address=url.full_url if isinstance(url,urllib.request.Request) else url
    port=urllib.parse.urlparse(address).port or 0
    body=url.data if isinstance(url,urllib.request.Request) else data
    size=len(body or b'');start=time.perf_counter();status=0;reset=False
    kind='blocked' if address.endswith('/api/annotations') else 'logs'
    try:
        response=ORIGINAL(url,data=data,timeout=timeout,**kwargs);status=response.status;return response
    except urllib.error.HTTPError as error:
        status=error.code;raise
    except ConnectionResetError:
        reset=True;raise
    finally:
        elapsed=(time.perf_counter()-start)*1000
        observe(kind,port,size,status,reset,elapsed)
        if kind=='blocked':DATA.update(port=port,body_bytes=size,status=status,reset=reset,request_ms=elapsed)

class ObservedServing(Serving):
    def __enter__(self):
        start=time.perf_counter();result=super().__enter__();elapsed=(time.perf_counter()-start)*1000
        DATA['startup_ms']=elapsed;DATA['ready']=True
        observe('ready',self.port,0,200,False,elapsed)
        return result
    def __exit__(self,*args):
        self.log.flush()
        Path(sys.argv[1]).with_suffix('.server.log').write_text(Path(str(self.database)+'.log').read_text(encoding='utf-8'),encoding='utf-8')
        return super().__exit__(*args)

def main():
    test_privacy.Serving=ObservedServing
    test_privacy.urllib=SimpleNamespace(request=SimpleNamespace(Request=urllib.request.Request,urlopen=test_request),error=urllib.error)
    failed=False;error_number=0
    start=time.perf_counter()
    try:
        test_privacy.Privacy('test_real_probe_intake_and_log_raw_copies').test_real_probe_intake_and_log_raw_copies()
    except ConnectionResetError as error:
        failed=True;error_number=error.winerror or 0
    DATA.update(failed=failed,error_number=error_number,test_ms=(time.perf_counter()-start)*1000)
    Path(sys.argv[1]).write_text(json.dumps(DATA,indent=2),encoding='utf-8')
    print('ROUND2_TEST '+json.dumps(DATA),flush=True)
    return dict(DATA)
