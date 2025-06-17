package database

import (
	"database/sql"
	"fmt"
)

func RunMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS core;`,

		// Tabela user
		`CREATE TABLE IF NOT EXISTS core.user (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			cpf VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true,
			inicial BOOLEAN DEFAULT false,
			dell BOOLEAN DEFAULT false,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela user_details
		`CREATE TABLE IF NOT EXISTS core.user_details (
			id UUID PRIMARY KEY,
			id_user UUID REFERENCES core.user(id),
			cpf_valid BOOLEAN DEFAULT false,
			email_valid BOOLEAN DEFAULT false,
			cep VARCHAR(70),
			telefone VARCHAR(70),
			apelido VARCHAR(200),
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela role
		`CREATE TABLE IF NOT EXISTS core.role (
			id SERIAL PRIMARY KEY,
			role_name VARCHAR(255) NOT NULL
		);`,

		// Inserção da role ROLE_OPERATOR
		`INSERT INTO core.role (role_name) VALUES ('ROLE_OPERATOR');`,
		
		// Inserção da role ROLE_ADMIN
		`INSERT INTO core.role (role_name) VALUES ('ROLE_ADMIN');`,
		

		// Tabela user_role
		`CREATE TABLE IF NOT EXISTS core.user_role (
			id_user UUID REFERENCES core.user(id),
			id_role INT8 REFERENCES core.role(id),
			PRIMARY KEY (id_user, id_role)
		);`,

		// Tabela user_login
		`CREATE TABLE IF NOT EXISTS core.user_login (
			id UUID PRIMARY KEY,
			email VARCHAR(255),
			id_user UUID REFERENCES core.user(id),
			pass_valid BOOLEAN DEFAULT false,
			date TIMESTAMP DEFAULT now()
		);`,

		// Tabela doacao
		`CREATE TABLE IF NOT EXISTS core.doacao (
			id UUID PRIMARY KEY,
			id_user UUID REFERENCES core.user(id),
			name VARCHAR(255) NOT NULL,
			valor DOUBLE PRECISION NOT NULL,
			active BOOLEAN DEFAULT true,
			dell BOOLEAN DEFAULT false,
			date_start TIMESTAMP,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela doacao_details
		`CREATE TABLE IF NOT EXISTS core.doacao_details (
			id UUID PRIMARY KEY,
			id_doacao UUID REFERENCES core.doacao(id),
			texto VARCHAR(255),
			img_caminho VARCHAR(255),
			area VARCHAR(255)
		);`,

		// Tabela doacao_qrcode
		`CREATE TABLE IF NOT EXISTS core.doacao_qrcode (
			id UUID PRIMARY KEY,
			id_doacao UUID REFERENCES core.doacao(id),
			qrcode VARCHAR(255),
			valor DOUBLE PRECISION NOT NULL,
			active BOOLEAN DEFAULT true,
			dell BOOLEAN DEFAULT false,
			date_start TIMESTAMP,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela doacao_pagametos
		`CREATE TABLE IF NOT EXISTS core.doacao_pagametos (
			id UUID PRIMARY KEY,
			indetificador VARCHAR NOT NULL,
			id_doacao UUID REFERENCES core.doacao(id),
			id_doacao_qrcode UUID REFERENCES core.doacao_qrcode(id),
			texto VARCHAR,
			valor DOUBLE PRECISION NOT NULL,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela saque_conta
		`CREATE TABLE IF NOT EXISTS core.saque_conta (
			id UUID PRIMARY KEY,
			id_user UUID REFERENCES core.user(id),
			banco VARCHAR(255),
			banco_nome VARCHAR(255),
			conta VARCHAR(255),
			agencia VARCHAR(255),
			digito VARCHAR(10),
			is_user BOOLEAN DEFAULT false,
			cpf VARCHAR(255),
			pix VARCHAR(255),
			telefone VARCHAR(255),
			endereco VARCHAR(255),
			active BOOLEAN DEFAULT true,
			dell BOOLEAN DEFAULT false,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela saque_details
		`CREATE TABLE IF NOT EXISTS core.saque_details (
			id UUID PRIMARY KEY,
			id_saque_conta UUID REFERENCES core.saque_conta(id),
			valor DOUBLE PRECISION NOT NULL,
			realizado BOOLEAN DEFAULT false,
			error VARCHAR(255),
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela com o link agradavel da doação 
		`CREATE TABLE IF NOT EXISTS core.doacao_link (
			id UUID PRIMARY KEY,
			id_doacao UUID NOT NULL,
			nome_link VARCHAR(255) NOT NULL,
			FOREIGN KEY (id_doacao) REFERENCES core.doacao (id)
		)`,

		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,

		`CREATE TABLE pix_qrcode (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_doacao UUID NOT NULL,
			valor NUMERIC(10,2) NOT NULL,
			cpf VARCHAR(14) NOT NULL,
			nome VARCHAR(255) NOT NULL,
			mensagem VARCHAR(255),
			anonimo BOOLEAN NOT NULL,
			visivel BOOLEAN NOT NULL DEFAULT FALSE,
			data_criacao TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT now()
		)`,


		`CREATE TABLE pix_qrcode_status (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_pix_qrcode UUID NOT NULL REFERENCES pix_qrcode(id) ON DELETE CASCADE,
			data_criacao TIMESTAMP WITHOUT TIME ZONE NOT NULL,
			expiracao INTEGER NOT NULL,
			tipo_pagamento VARCHAR(255),
			loc_id INTEGER,
			loc_tipo_cob VARCHAR(50),
			loc_criacao TIMESTAMP WITHOUT TIME ZONE,
			location TEXT,
			pix_copia_e_cola TEXT,
			chave VARCHAR(255),
			id_pix VARCHAR(255),
			status VARCHAR(50),
			buscar BOOLEAN NOT NULL DEFAULT FALSE,
			finalizado BOOLEAN NOT NULL DEFAULT FALSE,
			data_pago TIMESTAMP WITHOUT TIME ZONE DEFAULT NULL
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("erro ao executar a query: %v\n%v", err, query)
		}
	}

	return nil
}
