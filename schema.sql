CREATE TABLE throughput_record(
    id BIGSERIAL PRIMARY KEY,
    file_datetime timestamp NOT NULL,
    success boolean NOT NULL DEFAULT false,
    exec_time timestamp NOT NULL
)

CREATE TABLE throughput(
    storeid uuid NOT NULL,
    utc timestamp NOT NULL,
    requestsize bigint NOT NULL,
    responsesize bigint NOT NULL,
    PRIMARY KEY (storeid, utc)
)

CREATE TABLE throughput_old(
    cname varchar(64) NOT NULL,
    utc timestamp NOT NULL,
    requestsize bigint NOT NULL,
    responsesize bigint NOT NULL,
    PRIMARY KEY (cname, utc)
)