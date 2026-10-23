import json,subprocess,time,sys,datetime,hashlib
from pathlib import Path
REPO=Path(__file__).resolve().parents[2]
OUT=Path(__file__).resolve().parent
PY=REPO/'adapters/provenance-python/.venv/Scripts/python.exe'
ADAPTER=REPO/'adapters/provenance-python'
mode=sys.argv[1]
folder=OUT/mode
folder.mkdir(exist_ok=True)
if (folder/'results.json').exists(): raise SystemExit('refuse rerun/overwrite')
results=[]
for i in range(1,31):
 start=datetime.datetime.now(datetime.timezone.utc).isoformat();begin=time.perf_counter()
 cmd=[str(PY),'-m','unittest','discover','-s','tests','-p','test_privacy.py','-k','test_real_probe_intake_and_log_raw_copies','-v']
 cwd=ADAPTER
 if mode=='phase2':
  cmd=[str(PY),str(OUT/'scratch'/'run_test.py')];cwd=OUT/'scratch'
 p=subprocess.run(cmd,cwd=cwd,capture_output=True,timeout=60)
 text=(p.stdout+p.stderr).decode('utf-8',errors='replace')
 (folder/f'run-{i:02}.log').write_text(text,encoding='utf-8')
 row=dict(index=i,start_utc=start,duration_s=time.perf_counter()-begin,exit=p.returncode,reset10054='WinError 10054' in text,test_body_ran=('Ran 1 test' in text and 'Failed to import test module' not in text))
 results.append(row);(folder/'results.json').write_text(json.dumps(results,indent=2),encoding='utf-8')
 if i%5==0: print(mode,i,'fails',sum(r['exit']!=0 for r in results),flush=True)
print(json.dumps(dict(mode=mode,passes=sum(r['exit']==0 for r in results),fails=sum(r['exit']!=0 for r in results),resets=sum(r['reset10054'] for r in results),duration_s=sum(r['duration_s'] for r in results))))
