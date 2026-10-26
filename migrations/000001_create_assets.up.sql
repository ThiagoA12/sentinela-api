CREATE TABLE assets (
    id          uuid         PRIMARY KEY DEFAULT uuidv7(),
    name        varchar(120) NOT NULL,
    identifier  varchar(255) NOT NULL,
    type        text         NOT NULL    
                CHECK (type IN ('servidor', 'aplicacao', 'rede', 'estacao', 'nuvem')),
    criticality text         NOT NULL
                CHECK (criticality IN ('baixa', 'media', 'alta', 'critica')),
    created_at  timestamptz NOT NULL DEFAULT now()             
);