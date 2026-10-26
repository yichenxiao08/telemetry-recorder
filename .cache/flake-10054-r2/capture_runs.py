import json,subprocess,socket,time,datetime
from pathlib import Path
OUT=Path(__file__).resolve().parent;REPO=OUT.parents[1];ADAPTER=REPO/'adapters/provenance-python';PY=ADAPTER/'.venv/Scripts/python.exe';BIN=OUT/'signalscout.exe'
folder=OUT/'phase3';folder.mkdir(exist_ok=True)
if (folder/'results.json').exists():raise SystemExit('refuse overwrite')
with socket.socket() as s:s.bind(('127.0.0.1',0));port=s.getsockname()[1]
endpoint=f'http://127.0.0.1:{port}'
config={'version':1,'synthetic_only':True,'endpoint':endpoint,'service':'synthetic.privacy10054.r2','probes':[{'module':'r2_capture','function':'main','capture':{'return':True}},{'module':'r2_capture','function':'observe','capture':{'args':['kind','port','body_bytes','status','reset','elapsed_ms']}}]}
(folder/'config.json').write_text(json.dumps(config,indent=2),encoding='utf-8')
log=open(folder/'collector.log','w',encoding='utf-8')
server=subprocess.Popen([str(BIN),'--db',str(folder/'recordings.db'),'serve','--listen',f'127.0.0.1:{port}','--retention','0'],stdout=log,stderr=log)
(folder/'collector.json').write_text(json.dumps({'endpoint':endpoint,'pid':server.pid}),encoding='utf-8')
try:
 import urllib.request
 for attempt in range(150):
  try:
   with urllib.request.urlopen(endpoint+'/runs',timeout=.2):break
  except OSError:time.sleep(.02)
 else:raise RuntimeError('collector setup failed')
 rows=[]
 for i in range(1,21):
  report=folder/f'observation-{i:02}.json';doctor=folder/f'doctor-{i:02}.json'
  command=[str(PY),'-m','scout_probe','--config',str(folder/'config.json'),'--doctor',str(doctor),'--path',str(OUT),'--manifest','--drain-timeout','30','r2_capture:main','--',str(report)]
  begin=time.perf_counter();start=datetime.datetime.now(datetime.timezone.utc).isoformat()
  p=subprocess.run(command,cwd=ADAPTER,capture_output=True,timeout=60)
  (folder/f'capture-{i:02}.log').write_bytes(p.stdout+p.stderr)
  observation=json.loads(report.read_text()) if report.exists() else None
  row={'index':i,'utc':start,'duration_s':time.perf_counter()-begin,'exit':p.returncode,'observation':observation}
  rows.append(row);(folder/'results.json').write_text(json.dumps(rows,indent=2),encoding='utf-8')
  print('recording',i,'exit',p.returncode,'observation',observation,flush=True)
  if observation is None:raise RuntimeError('recording setup failed; stop instead of spending remaining budget')
  if sum(r['observation']['failed'] for r in rows)>=5 and sum(not r['observation']['failed'] for r in rows)>=5:break
 # Snapshot while writer remains alive, for later CLI-only analysis.
 (folder/'runs-list.json').write_bytes(subprocess.check_output([str(BIN),'--db',str(folder/'recordings.db'),'runs','list','--json']))
 # Keep collector alive for cohort CLI work; detached from this Python process.
 print('collector left running',server.pid,flush=True)
except BaseException:
 server.terminate();server.wait();raise
finally:log.close()
