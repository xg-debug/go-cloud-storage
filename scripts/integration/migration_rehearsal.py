"""Offline migration rehearsal, only called inside the disposable smoke project.
Uses the actual legacy DDL and explicit column lists. Never takes production DSNs.
"""
from pathlib import Path

TABLES = {
    'user': 'user', 'file': 'file', 'storage_quota': 'file',
    'favorite': 'file', 'recycle_bin': 'file', 'share': 'share',
    'notification': 'notification',
}
SOURCE = 'cloudbox_rehearsal_legacy'


def run(sql, stop, start, roles, verify):
    print('Rehearsing offline legacy-to-split migration in disposable databases...', flush=True)
    for role in reversed(roles):
        stop(role)
    assert sql('cloudbox_file','SELECT COUNT(*) FROM object_deletion').stdout.strip()=='0', 'pending object cleanup must not be dropped'
    sql('mysql', f'CREATE DATABASE `{SOURCE}`')
    ddl = (Path(__file__).resolve().parents[2] / 'db.sql').read_text()
    sql(SOURCE, ddl)

    def columns(database, table):
        query = f"SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='{database}' AND TABLE_NAME='{table}' AND GENERATION_EXPRESSION = '' ORDER BY ORDINAL_POSITION"
        return sql(database, query).stdout.strip().splitlines()

    fields = {}
    # Seed the legacy schema with the real API-created fixture, preserving IDs,
    # object keys, password hashes and relationships; disable FKs only for order.
    for table, role in TABLES.items():
        target = 'cloudbox_' + role
        names = [c for c in columns(target, table) if c in columns(SOURCE, table)]
        fields[table] = ','.join('`'+name+'`' for name in names)
        cols = fields[table]
        sql(SOURCE, f'SET FOREIGN_KEY_CHECKS=0; INSERT INTO `{table}` ({cols}) SELECT {cols} FROM `{target}`.`{table}`; SET FOREIGN_KEY_CHECKS=1;')

    date_precision = {}
    for table in TABLES:
        rows = sql(SOURCE, f"SELECT COLUMN_NAME,DATETIME_PRECISION FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='{SOURCE}' AND TABLE_NAME='{table}' AND DATA_TYPE IN ('datetime','timestamp')").stdout.strip().splitlines()
        date_precision[table] = {row.split('\t')[0]:int(row.split('\t')[1]) for row in rows}

    def projection(table):
        # Compare every migrated field at the precision representable by legacy.
        # HEX avoids delimiters and newlines inside user-controlled text.
        expressions = []
        for field in fields[table].split(','):
            precision = date_precision[table].get(field.strip('`'))
            expressions.append(field if precision is None else f"LEFT(DATE_FORMAT({field},'%Y-%m-%d %H:%i:%s.%f'),{19 if precision==0 else 20+precision})")
        return ','.join(expressions)

    def snapshot(database, table):
        cols = projection(table)
        return sql(database, f'SELECT HEX(CAST(JSON_ARRAY({cols}) AS CHAR)) FROM `{table}` ORDER BY {"user_id" if table=="storage_quota" else "id"}').stdout

    expected = {table:snapshot(SOURCE,table) for table in TABLES}
    for table, role in TABLES.items():
        sql('cloudbox_'+role, f'DELETE FROM `{table}`')
    # No historical events are replayed. Source remains untouched for rollback.
    for role in ['file','share','notification']:
        for table in (['event_outbox'] if role=='file' else ['event_outbox','event_inbox'] if role=='share' else ['event_inbox']):
            sql('cloudbox_'+role, f'DELETE FROM `{table}`')
    statements = []
    for table,role in TABLES.items():
        cols = fields[table]
        statements.append(f'INSERT INTO `cloudbox_{role}`.`{table}` ({cols}) SELECT {cols} FROM `{SOURCE}`.`{table}`')
    sql(SOURCE, 'START TRANSACTION;'+ ';'.join(statements) + ';COMMIT;')
    for table,role in TABLES.items():
        assert snapshot('cloudbox_'+role,table)==expected[table], f'migration changed {table}'
    assert sql('cloudbox_file', 'SELECT COUNT(*) FROM file f LEFT JOIN file p ON f.parent_id=p.id WHERE f.parent_id IS NOT NULL AND p.id IS NULL').stdout.strip()=='0'
    assert sql('cloudbox_file', 'SELECT COUNT(*) FROM storage_quota q WHERE q.used != (SELECT COALESCE(SUM(size),0) FROM file f WHERE f.user_id=q.user_id AND f.is_dir=0)').stdout.strip()=='0'
    for role in roles:
        start(role)
    verify()
    # Simulate rollback after new writes: take current split data back into a
    # separate legacy-shaped copy, retain the source snapshot for comparison.
    for role in reversed(roles):
        stop(role)
    rollback = 'cloudbox_rehearsal_rollback'
    sql('mysql',f'CREATE DATABASE `{rollback}`')
    sql(rollback,ddl)
    for table,role in TABLES.items():
        cols=fields[table]
        sql(rollback, f'SET FOREIGN_KEY_CHECKS=0; INSERT INTO `{table}` ({cols}) SELECT {projection(table)} FROM `cloudbox_{role}`.`{table}`; SET FOREIGN_KEY_CHECKS=1;')
        assert snapshot(rollback,table)==snapshot('cloudbox_'+role,table), f'reverse copy changed {table}'
    # Data relationships and content-key coverage, including new writes.
    assert sql(rollback,'SELECT COUNT(*) FROM file f LEFT JOIN user u ON f.user_id=u.id WHERE u.id IS NULL').stdout.strip()=='0'
    assert sql(rollback,'SELECT COUNT(*) FROM share s LEFT JOIN file f ON s.file_id=f.id WHERE f.id IS NULL').stdout.strip()=='0'
    assert sql(rollback,'SELECT COUNT(*) FROM file f JOIN cloudbox_file.file t ON BINARY t.id=BINARY f.id WHERE NOT(BINARY f.oss_object_key <=> BINARY t.oss_object_key) OR NOT(BINARY f.file_hash <=> BINARY t.file_hash) OR f.size != t.size').stdout.strip()=='0'
    for table in TABLES:
        assert snapshot(SOURCE,table)==expected[table], 'source changed during migration'
    for role in roles:
        start(role)
    print('PASS: legacy DDL import, row-by-row forward comparison, business checks and reverse-copy rehearsal (not a production rollback).', flush=True)
