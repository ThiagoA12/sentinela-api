INSERT INTO assets (name, identifier, type, criticality, owner_id, created_by)
SELECT v.name, v.identifier, v.type, v.criticality, adm.id, adm.id
FROM (VALUES
    ('Servidor de banco de dados', 'db-prod-01.interno', 'servidor',  'critica'),
    ('Portal do aluno',            'portal.exemplo.edu', 'aplicacao', 'alta'),
    ('Estação da secretaria',      'sec-ws-07',          'estacao',   'media')
) AS v(name, identifier, type, criticality)
LEFT JOIN (
    SELECT id FROM users WHERE role = 'admin' ORDER BY created_at, id LIMIT 1
) AS adm ON true
WHERE NOT EXISTS (
    SELECT 1 FROM assets a WHERE a.identifier = v.identifier
);