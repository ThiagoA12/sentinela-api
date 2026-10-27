-- Dados de exemplo para desenvolvimento. Nunca rodar em produção.
-- Pode ser executado várias vezes: só insere o que ainda não existe.
INSERT INTO assets (name, identifier, type, criticality)
SELECT v.name, v.identifier, v.type, v.criticality
FROM (VALUES
    ('Servidor de banco de dados', 'db-prod-01.interno', 'servidor',  'critica'),
    ('Portal do aluno',            'portal.exemplo.edu', 'aplicacao', 'alta'),
    ('Estação da secretaria',      'sec-ws-07',          'estacao',   'media')
) AS v(name, identifier, type, criticality)
WHERE NOT EXISTS (
    SELECT 1 FROM assets a WHERE a.identifier = v.identifier
);