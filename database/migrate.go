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
			id_tipo_conta UUID,  -- nova coluna
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
			img_perfil VARCHAR(255),
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
			closed BOOLEAN DEFAULT false,
			date_start TIMESTAMP,
			date_create TIMESTAMP DEFAULT now(),
			date_update TIMESTAMP DEFAULT now()
		);`,

		// Tabela doacao_details
		`CREATE TABLE IF NOT EXISTS core.doacao_details (
			id UUID PRIMARY KEY,
			id_doacao UUID REFERENCES core.doacao(id),
			texto TEXT CHECK (length(texto) <= 5500),
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
		`CREATE TABLE IF NOT EXISTS core.doacao_pagamentos (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_doacao UUID NOT NULL REFERENCES core.doacao(id) ON DELETE CASCADE,
			valor_disponivel NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
			valor_tranferido NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
			data_tranferido TIMESTAMP,
			solicitado BOOLEAN NOT NULL DEFAULT false,
			data_solicitado TIMESTAMP,
			status VARCHAR(255),
			img VARCHAR(255),
			pdf VARCHAR(255),
			banco VARCHAR(255),
			conta VARCHAR(255),
			agencia VARCHAR(255),
			digito VARCHAR(255),
			pix VARCHAR(255),
			data_update TIMESTAMP DEFAULT now()
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

		`CREATE TABLE IF NOT EXISTS core.pix_qrcode (
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

		`CREATE TABLE IF NOT EXISTS core.pix_qrcode_status (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_pix_qrcode UUID NOT NULL REFERENCES core.pix_qrcode(id) ON DELETE CASCADE,
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

		`CREATE TABLE IF NOT EXISTS core.conta_nivel (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_user UUID NOT NULL REFERENCES core.user(id),
			nivel VARCHAR(100) NOT NULL,
			ativo BOOLEAN DEFAULT false,
			status VARCHAR(100),
			data_pagamento TIMESTAMP,
			tipo_pagamento VARCHAR(100),
			data_update TIMESTAMP DEFAULT now()
		);`,

		`CREATE TABLE IF NOT EXISTS core.conta_nivel_pagamento (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_user UUID NOT NULL REFERENCES core.user(id),
			pago_data TIMESTAMP,
			pago BOOLEAN DEFAULT false,
			valor NUMERIC(10,2),
			status VARCHAR(100),
			codigo VARCHAR(255),
			data_create TIMESTAMP DEFAULT now(),
			referente VARCHAR(255),
			valido BOOLEAN DEFAULT true,
			txid VARCHAR(255),
			pg_status VARCHAR(100),
			cpf VARCHAR(20),
			chave VARCHAR(255),
			pixCopiaECola TEXT,
			expiracao INTEGER
		);`,

		`CREATE TABLE IF NOT EXISTS core.visualization (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_doacao UUID NOT NULL REFERENCES core.doacao(id) ON DELETE CASCADE,
			visualization INTEGER DEFAULT 0,
			donation_like INTEGER DEFAULT 0,
			love INTEGER DEFAULT 0,
			shared INTEGER DEFAULT 0,
			acesse_donation INTEGER DEFAULT 0,
			create_pix INTEGER DEFAULT 0,
			create_cartao INTEGER DEFAULT 0,
			create_paypal INTEGER DEFAULT 0,
			create_google INTEGER DEFAULT 0,
			create_pag1 INTEGER DEFAULT 0,
			create_pag2 INTEGER DEFAULT 0,
			create_pag3 INTEGER DEFAULT 0,
			date_create TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
			date_update TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
		);`,

		`CREATE TABLE IF NOT EXISTS core.visualization_dth (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			id_visualization UUID NOT NULL REFERENCES core.visualization(id) ON DELETE CASCADE,
			ip VARCHAR(100),
			id_user UUID,
			idioma VARCHAR(50),
			tema VARCHAR(50),
			form VARCHAR(100),
			google VARCHAR(100),
			google_maps VARCHAR(100),
			google_ads VARCHAR(100),
			meta_pixel VARCHAR(100),
			Cookies_Stripe VARCHAR(100),
			Cookies_PayPal VARCHAR(100),
			visitor_info1_live VARCHAR(100),
			-- Ações do usuário
			donation_like BOOLEAN DEFAULT false,
			love BOOLEAN DEFAULT false,
			shared BOOLEAN DEFAULT false,
			acesse_donation BOOLEAN DEFAULT false,
			create_pix BOOLEAN DEFAULT false,
			create_cartao BOOLEAN DEFAULT false,
			create_paypal BOOLEAN DEFAULT false,
			create_google BOOLEAN DEFAULT false,
			create_pag1 BOOLEAN DEFAULT false,
			create_pag2 BOOLEAN DEFAULT false,
			create_pag3 BOOLEAN DEFAULT false,
			date_create TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("erro ao executar a query: %v\n%v", err, query)
		}
	}

	return nil
}
