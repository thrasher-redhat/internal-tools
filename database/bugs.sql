CREATE TABLE IF NOT EXISTS bugs (
    id              integer NOT NULL,
    component       text NOT NULL,
    target_release  text NOT NULL,
    assigned_to     text NOT NULL,
    status          text NOT NULL,
    summary         text NOT NULL,
    keywords        text[] NOT NULL,
    cf_pm_score     integer NOT NULL,
    externals       jsonb NOT NULL,
    datestamp       date NOT NULL,
    PRIMARY KEY (id, datestamp)
);

