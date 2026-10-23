import json,subprocess,time,urllib.request
from pathlib import Path
OUT=Path(__file__).resolve().parent;folder=OUT/'phase3';BIN=OUT/'signalscout.exe';db=folder/'recordings.db';endpoint=json.loads((folder/'collector.json').read_text())['endpoint'];port=endpoint.rsplit(':',1)[1]
commands=json.loads((OUT/'analysis-cli-ledger.json').read_text()) if (OUT/'analysis-cli-ledger.json').exists() else []
def cli(name,args):
 cmd=[str(BIN),'--db',str(db),*args];p=subprocess.run(cmd,capture_output=True,timeout=60);commands.append({'name':name,'args':args,'exit':p.returncode});(folder/(name+'.json')).write_bytes(p.stdout);(folder/(name+'.stderr.txt')).write_bytes(p.stderr);(OUT/'analysis-cli-ledger.json').write_text(json.dumps(commands,indent=2),encoding='utf-8')
 if p.returncode:raise RuntimeError(name+': '+p.stderr.decode())
 return json.loads(p.stdout)
log=open(folder/'collector-analysis.log','w',encoding='utf-8');server=subprocess.Popen([str(BIN),'--db',str(db),'serve','--listen','127.0.0.1:'+port,'--retention','0'],stdout=log,stderr=log)
try:
 for i in range(150):
  try:
   with urllib.request.urlopen(endpoint+'/runs',timeout=.2):break
  except OSError:time.sleep(.02)
 runs=cli('runs-list',['runs','list'])
 print('runs',len(runs),flush=True)
 completeness=[]
 for i in range(1,20):
  doctor=json.loads((folder/f'doctor-{i:02}.json').read_text());epoch=doctor['counters']['manifest']['epoch']
  c=cli(f'completeness-{i:02}',['cohort','completeness','--scope',epoch]);completeness.append(c)
 (folder/'all-completeness.json').write_text(json.dumps(completeness,indent=2),encoding='utf-8')
 profile={'id':'flake10054-r2','unit':'per-run','workload':'r2_capture.main','step':'r2_capture.main','envelope_profile':'synthetic','envelope_version':'1','numeric':[{'name':n,'selector':'output.'+n,'unit':u,'epsilon':eps,'floor':1} for n,u,eps in [('startup_ms','ms',1),('request_ms','ms',1),('test_ms','ms',1),('body_bytes','bytes',1),('port','port',1)]],'categories':[{'name':n,'selector':'output.'+n} for n in ['ready','reset','status']]}
 (folder/'analysis-profile.json').write_text(json.dumps(profile,indent=2),encoding='utf-8')
 for label,value in [('failure','true'),('passing','false')]:
  result=cli('cohort-'+label,['cohort','create','r2-'+label,'--endpoint',endpoint,'--profile',str(folder/'analysis-profile.json'),'--from','2026-10-23T20:00:00Z','--to','2026-10-24T00:00:00Z','--limit','100','--predicate','output.failed','--equals',value,'--require-complete'])
  print(label,'selected',len(result.get('run_ids',[])),flush=True)
 result=cli('explain',['explain-cohort','r2-failure','r2-passing','--json']);print('explain status',result.get('status'),flush=True)
finally:
 server.terminate();server.wait(timeout=15);log.close()
