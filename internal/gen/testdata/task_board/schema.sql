CREATE TABLE boards (b_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, b_dt_created DATE NOT NULL, b_dt_updated DATE NOT NULL, b_name STRING NOT NULL);

CREATE TABLE tasks (t_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, t_dt_created DATE NOT NULL, t_dt_updated DATE NOT NULL, t_subject STRING NOT NULL, t_comments TEXT, t_status BOOLEAN NOT NULL DEFAULT (0), b_id BIGINT NOT NULL, FOREIGN KEY (b_id) REFERENCES boards (b_id) ON DELETE CASCADE);

CREATE TABLE _no_ai_pk (id BIGINT PRIMARY KEY NOT NULL, name TEXT NOT NULL);

CREATE TABLE users (u_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, u_name TEXT NOT NULL);

CREATE TABLE task_users (
    t_id       INTEGER NOT NULL
                       REFERENCES tasks (t_id),
    u_id       INTEGER REFERENCES users (u_id) 
                       NOT NULL,
    tu_comment TEXT,
    tu_date TEXT,
    PRIMARY KEY (t_id, u_id)
);

