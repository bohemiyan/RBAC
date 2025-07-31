CREATE TABLE permissions (
    permission_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE roles (
    role_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE role_permissions (
    role_id INTEGER REFERENCES roles(role_id),
    permission_id INTEGER REFERENCES permissions(permission_id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE employee_roles (
    employee_id INTEGER NOT NULL,
    role_id INTEGER REFERENCES roles(role_id),
    PRIMARY KEY (employee_id, role_id)
);

CREATE TABLE audits (
    employee_id INTEGER,
    action VARCHAR(255),
    resource TEXT,
    success BOOLEAN,
    timestamp TIMESTAMP
);