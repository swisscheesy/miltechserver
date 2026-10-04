#!/usr/bin/env python3
"""No network: private runner copies and a recording psql executable."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
RUNNER = ROOT / "scripts/apply-shops-remediation-migrations.sh"

class RunnerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        (self.root / "scripts").mkdir()
        shutil.copytree(ROOT / "migrations", self.root / "migrations")
        self.runner = self.root / "scripts" / RUNNER.name
        self.runner.write_bytes(RUNNER.read_bytes())
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.calls = self.root / "calls.jsonl"
        fake = self.bin / "psql"
        fake.write_text("#!/usr/bin/env python3\nimport json,os,sys\nwith open(os.environ['CALLS'],'a') as f: f.write(json.dumps({'args':sys.argv[1:],'sql':sys.stdin.read()})+'\\n')\nprint('SQL_APPLIED_GENERATION_PENDING|123|123')\nsys.exit(int(os.environ.get('PSQL_EXIT','0')))\n")
        fake.chmod(0o700)
        self.env = {k:v for k,v in os.environ.items() if not k.startswith(('PG','JET_','DB_','SHOPS_RELEASE_'))}
        self.env.update(PATH=str(self.bin)+os.pathsep+os.environ['PATH'], CALLS=str(self.calls), PGSERVICE='approved_test', DB_USERNAME='application')
        self.proof=self.root/'generation.json'
        self.args = ['miltech_ng_test','forward-019','approval-test','127.0.0.1','5432','migration_owner','a'*64,str(self.proof)]

    def pin(self,stage="019"):
        pmcs={'.gen/miltech_ng/public/model/user_pmcs_'+str(n)+'.go': 'c'*64 for n in range(32)}
        proof={'stage':str(int(stage)-1).zfill(3),'source_sha256':'d'*64,'generation_manifest':{'database':'miltech_ng_test','address':'127.0.0.1','port':5432,'user':'application','schema':'public','catalog_sha256':'a'*64},'pmcs_before':pmcs,'pmcs_after':pmcs,'generated_packages_compile':'PASS'}
        self.proof.write_text(json.dumps(proof))
        record = dict(address='127.0.0.1',port=5432,role='migration_owner',version=140018,application_role='application',service='approved_test',catalog_sha256='a'*64,data_sha256='b'*64,authorization='approval-test',generation_evidence_sha256=hashlib.sha256(self.proof.read_bytes()).hexdigest(),source_sha256='d'*64,pmcs_sha256=hashlib.sha256(json.dumps(pmcs,sort_keys=True).encode()).hexdigest(),probe_shop_id='fixture-shop',probe_user_id='fixture-user',test_target_evidence='test-first')
        pins = {'miltech_ng_test': {stage: record}, 'miltech_ng': {}}
        text = self.runner.read_text()
        start = text.index('TARGET_PINS = ')
        end = text.index('\n',start)
        self.runner.write_text(text[:start]+'TARGET_PINS = '+repr(pins)+text[end:])

    def run_runner(self):
        return subprocess.run(['bash',str(self.runner),*self.args],env=self.env,text=True,capture_output=True)

    def records(self):
        return [json.loads(x) for x in self.calls.read_text().splitlines()] if self.calls.exists() else []

    def test_runner_refuses_unpinned_and_wrong_target(self):
        self.assertNotEqual(self.run_runner().returncode,0)
        connection_attempts_for_unpinned = len(self.records())
        self.assertEqual(connection_attempts_for_unpinned,0)
        self.args[0]='other_database'
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_runner_uses_same_identity_session(self):
        self.pin()
        result=self.run_runner()
        self.assertEqual(result.returncode,0,result.stderr)
        identity_session_id,migration_session_id=[int(x.split('=')[1]) for x in result.stdout.splitlines()[0].split()]
        self.assertEqual(identity_session_id,migration_session_id)
        records=self.records()
        self.assertEqual(len(records),1)
        sql=records[0]['sql']
        self.assertLess(sql.index('BEGIN;'),sql.index('current_database()'))
        self.assertLess(sql.index('current_database()'),sql.index('-- BEGIN REVIEWED MIGRATION'))
        self.assertLess(sql.index('catalog_sha256'),sql.index('-- BEGIN REVIEWED MIGRATION'))
        self.assertIn('pg_backend_pid()',sql)
        self.assertIn('identity_session_id',sql)
        self.assertIn('migration_session_id',sql)
        self.assertIn('SET LOCAL ROLE',sql)
        self.assertIn('INSERT INTO public.shop_messages',sql)
        self.assertIn('ROLLBACK',sql)
        self.assertIn('STOP: tagged generation',result.stdout)

    def test_asset_reverse_locks_all_proof_before_checks_and_pin_matches(self):
        path = ROOT / 'migrations/020_rollback_shop_message_asset_lifecycle.sql'
        sql = path.read_text()
        boundary = sql.index('IF EXISTS')
        positions = [sql.index('LOCK TABLE public.' + table + ' IN ACCESS EXCLUSIVE MODE;')
                     for table in ('shop_message_uploads', 'shop_message_asset_references', 'shop_message_blob_cleanup_jobs')]
        self.assertEqual(positions, sorted(positions))
        self.assertTrue(all(position < boundary for position in positions))
        self.assertLess(sql.index("SET LOCAL lock_timeout = '5s';"), positions[0])
        self.assertIn(hashlib.sha256(path.read_bytes()).hexdigest(), RUNNER.read_text())

    def test_all_forward_envelopes_are_supported(self):
        for stage in ('019','020','021','022','023'):
            with self.subTest(stage=stage):
                self.pin(stage)
                self.args[1]='forward-'+stage
                result=self.run_runner()
                self.assertEqual(result.returncode,0,result.stderr)

    def test_inherited_selectors_refuse_before_contact(self):
        self.pin()
        for name in ('PGHOST','PGHOSTADDR','PGPORT','PGUSER','PGDATABASE','PGOPTIONS','PGTARGETSESSIONATTRS','JET_DSN'):
            with self.subTest(name=name):
                self.env[name]=''
                self.assertNotEqual(self.run_runner().returncode,0)
                self.assertEqual(len(self.records()),0)
                del self.env[name]

    def test_wrong_identity_authorization_and_role_refuse_before_contact(self):
        self.pin()
        for index in (2,3,4,5,6):
            original=self.args[index]
            self.args[index]='wrong'
            self.assertNotEqual(self.run_runner().returncode,0)
            self.assertEqual(len(self.records()),0)
            self.args[index]=original
        self.env['DB_USERNAME']='postgres'
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_changed_or_redirected_sql_refuses_before_contact(self):
        self.pin()
        path=self.root/'migrations/019_fix_shop_message_allocator_lock_order.sql'
        original=path.read_bytes()
        path.write_bytes(original+b'-- changed')
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)
        path.unlink()
        path.symlink_to(ROOT/'migrations/019_fix_shop_message_allocator_lock_order.sql')
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_generation_semantics_checked_even_with_matching_file_digest(self):
        for field,value in (('stage','017'),('source_sha256','e'*64),('generated_packages_compile','FAIL')):
            with self.subTest(field=field):
                self.pin()
                proof=json.loads(self.proof.read_text())
                proof[field]=value
                before=hashlib.sha256(self.proof.read_bytes()).hexdigest()
                self.proof.write_text(json.dumps(proof))
                after=hashlib.sha256(self.proof.read_bytes()).hexdigest()
                self.runner.write_text(self.runner.read_text().replace(before,after))
                self.assertNotEqual(self.run_runner().returncode,0)
                self.assertEqual(len(self.records()),0)

    def test_wrong_generation_target_and_changed_pmcs_refuse(self):
        for kind in ('database','user','catalog_sha256','pmcs_after'):
            with self.subTest(kind=kind):
                self.pin()
                before=hashlib.sha256(self.proof.read_bytes()).hexdigest()
                proof=json.loads(self.proof.read_text())
                if kind=='pmcs_after': proof[kind]={}
                else: proof['generation_manifest'][kind]='wrong'
                self.proof.write_text(json.dumps(proof))
                after=hashlib.sha256(self.proof.read_bytes()).hexdigest()
                self.runner.write_text(self.runner.read_text().replace(before,after))
                self.assertNotEqual(self.run_runner().returncode,0)
                self.assertEqual(len(self.records()),0)

    def test_missing_or_mismatched_generation_refuses_before_contact(self):
        self.pin()
        self.proof.write_text('{}')
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)
        self.proof.unlink()
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_disposable_mode_cannot_contact_named_target(self):
        self.pin()
        self.args.insert(0,'--disposable-rehearsal')
        self.env['SHOPS_RELEASE_TEST_MARKER']='f'*64
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_reverse_requires_separate_resolution_authorization(self):
        self.pin()
        self.args[1]='reverse-019'
        self.assertNotEqual(self.run_runner().returncode,0)
        self.assertEqual(len(self.records()),0)

    def test_session_failure_never_reports_applied(self):
        self.pin()
        self.env['PSQL_EXIT']='3'
        result=self.run_runner()
        self.assertNotEqual(result.returncode,0)
        self.assertNotIn('STOP: tagged generation',result.stdout)
        self.assertEqual(len(self.records()),1)


def rehearse_disposable():
    """Called only inside the existing protected wrapper; no named target aliases."""
    import ast
    import ipaddress
    import re
    import sys
    root, instance = (Path(p).resolve() for p in sys.argv[2:4])
    database,port,stage=sys.argv[4:7]
    marker=os.environ.get('SHOPS_RELEASE_TEST_MARKER','')
    if not re.fullmatch(r'miltech_test_[a-z0-9_]+',database) or not re.fullmatch(r'[0-9a-f]{64}',marker):
        raise SystemExit('Protected disposable wrapper context required')
    env={k:v for k,v in os.environ.items() if not k.startswith(('PG','JET_','DB_'))}
    def query(sql):
        result=subprocess.run(['psql','-X','-w','-At','-v','ON_ERROR_STOP=1','-h','127.0.0.1','-p',port,'-U','postgres','-d',database],input=sql,text=True,capture_output=True,env=env)
        if result.returncode: raise SystemExit('Disposable release runner fixture query failed')
        return result.stdout.strip()
    identity=query("SELECT current_database()||'|'||host(inet_server_addr())||'|'||inet_server_port()||'|'||current_user||'|'||marker FROM test_infrastructure.disposable_instance WHERE singleton=true;")
    if identity!='|'.join((database,'127.0.0.1',port,'postgres',marker)):
        raise SystemExit('Disposable runner identity/marker mismatch')
    content=(root/'scripts'/RUNNER.name).read_text()
    python=content.split("<<'PYTHON'\n",1)[1].rsplit('\nPYTHON',1)[0]
    constants={}
    for node in ast.parse(python).body:
        if isinstance(node,ast.Assign) and len(node.targets)==1 and isinstance(node.targets[0],ast.Name) and node.targets[0].id in ('DATA_SQL','CATALOG_SQL'):
            constants[node.targets[0].id]=ast.literal_eval(node.value)
    catalog=query("SELECT encode(sha256(convert_to(("+constants['CATALOG_SQL']+"),'UTF8')),'hex');")
    data=query("SELECT encode(sha256(convert_to(("+constants['DATA_SQL']+"),'UTF8')),'hex');")
    manifest=json.loads((instance/'source/.gen/miltech_ng/public/generation-manifest.json').read_text())
    pmcs=json.loads((instance/'pmcs-manifest.json').read_text())
    if manifest['catalog_sha256']!=catalog:
        raise SystemExit('Tagged catalog and runner catalog differ')
    source=hashlib.sha256((instance/'source-manifest.json').read_bytes()).hexdigest()
    proof={'stage':str(int(stage)-1).zfill(3),'source_sha256':source,'generation_manifest':manifest,'pmcs_before':pmcs,'pmcs_after':pmcs,'generated_packages_compile':'PASS'}
    proof_path=instance/'release-generation.json'
    proof_path.write_text(json.dumps(proof))
    pin=dict(address='127.0.0.1',port=int(port),role='postgres',version=int(query("SELECT current_setting('server_version_num');")),application_role='postgres',service='release_disposable',catalog_sha256=catalog,data_sha256=data,authorization='wrapper-owned-disposable',generation_evidence_sha256=hashlib.sha256(proof_path.read_bytes()).hexdigest(),source_sha256=source,pmcs_sha256=hashlib.sha256(json.dumps(pmcs,sort_keys=True).encode()).hexdigest(),probe_shop_id='release-shop',probe_user_id='release-user',test_target_evidence='protected-disposable-only')
    private=instance/'release-runner-source'
    (private/'scripts').mkdir(parents=True,exist_ok=True)
    if not (private/'migrations').exists(): shutil.copytree(root/'migrations',private/'migrations')
    start=content.index('TARGET_PINS = ')
    end=content.index('\n',start)
    content=content[:start]+'TARGET_PINS = '+repr({database:{stage:pin}})+content[end:]
    runner=private/'scripts'/RUNNER.name
    runner.write_text(content)
    service=instance/'release-services.conf'
    service.write_text('[release_disposable]\nhost=127.0.0.1\nport='+port+'\ndbname='+database+'\nuser=postgres\nsslmode=disable\nconnect_timeout=5\n')
    service.chmod(0o600)
    env.update(PGSERVICE='release_disposable',PGSERVICEFILE=str(service),DB_USERNAME='postgres')
    args=['bash',str(runner),'--disposable-rehearsal',database,'forward-'+stage,'wrapper-owned-disposable','127.0.0.1',port,'postgres',catalog,str(proof_path)]
    # Before each physical action, prove both checkpoint refusal paths use zero
    # psql calls. A failing recorder shadows psql without contacting PostgreSQL.
    recorder=instance/'release-refusal-bin'
    recorder.mkdir(exist_ok=True)
    calls=instance/'release-refusal-calls'
    fake=recorder/'psql'
    fake.write_text('#!/bin/sh\nprintf call >> "'+str(calls)+'"\nexit 90\n')
    fake.chmod(0o700)
    refused_env=dict(env,PATH=str(recorder)+os.pathsep+env['PATH'])
    raw=proof_path.read_bytes()
    for bad in (b'{}',None):
        if bad is None: proof_path.unlink()
        else: proof_path.write_bytes(bad)
        refused=subprocess.run(args,text=True,capture_output=True,env=refused_env)
        if refused.returncode==0 or calls.exists():
            raise SystemExit('Generation refusal contacted PostgreSQL')
    proof_path.write_bytes(raw)
    before=query('SELECT count(*) FROM shop_messages;')
    result=subprocess.run(args,text=True,capture_output=True,env=env)
    if result.returncode:
        # Fixture identifiers contain no credentials. Keep errors bounded and
        # never emit service contents or SQL input even on a failed rehearsal.
        raise SystemExit('Disposable runner failed: '+result.stderr.strip())
    if 'Generation is PENDING' not in result.stdout: raise SystemExit('Missing pending-generation receipt')
    match=re.search(r'identity_session_id=(\d+) migration_session_id=(\d+)',result.stdout)
    if not match or match[1]!=match[2]: raise SystemExit('Physical session identity mismatch')
    if query('SELECT count(*) FROM shop_messages;')!=before:
        raise SystemExit('Legacy application proof did not roll back')
    print('Release runner '+stage+' PASS: identity_session_id='+match[1]+' migration_session_id='+match[2]+'; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0')

if __name__=='__main__':
    import sys
    if len(sys.argv)>1 and sys.argv[1]=='--rehearse-disposable': rehearse_disposable()
    else: unittest.main()
