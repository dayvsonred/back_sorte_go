package models

type DoacaoDetails struct {
	ID        string `json:"id" db:"id"`
	IDDoacao  string `json:"id_doacao" db:"id_doacao"`
	Texto     string `json:"texto" db:"texto"`
	ImgCaminho string `json:"img_caminho" db:"img_caminho"`
	Area      string `json:"area" db:"area"`
}
