CREATE TABLE inventory (sku TEXT PRIMARY KEY, available INTEGER NOT NULL);
INSERT INTO inventory SELECT 'sku-' || lpad(n::text,3,'0'),1000000 FROM generate_series(0,999) n;
