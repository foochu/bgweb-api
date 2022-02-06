package met

import "encoding/xml"

type METData struct {
	XMLName      xml.Name            `xml:"met"`
	Info         METInfo             `xml:"info"`
	PreCrawford  PreCrawfordTable    `xml:"pre-crawford-table"`
	PostCrawford []PostCrawfordTable `xml:"post-crawford-table"`
	// /* Pre crawford MET */
	//  aarMET [MAXSCORE][MAXSCORE]float32
	//  mpPreCrawford METParameters
	// /* post-crawford MET */
	//  aarMETPostCrawford [2][MAXSCORE]float32
	//  ampPostCrawford [2]METParameters
}

type METInfo struct {
	XMLName     xml.Name `xml:"info"`
	Name        string   `xml:"name"`        /* Name of match equity table */
	Description string   `xml:"description"` /* Description of met */
	Length      int      `xml:"length"`      /* native length of met, -1 : pure calculated table */
	FileName    string   /* File name of met */
}

type PreCrawfordTable struct {
	XMLName xml.Name `xml:"pre-crawford-table"`
	Type    string   `xml:"type,attr"`
	Rows    []struct {
		ME []float32 `xml:"me"`
	} `xml:"row"`
	Parameters METParameters `xml:"parameters"`
}

type PostCrawfordTable struct {
	XMLName xml.Name `xml:"post-crawford-table"`
	Type    string   `xml:"type,attr"`
	Player  string   `xml:"player,attr"`
	Row     struct {
		ME []float32 `xml:"me"`
	} `xml:"row"`
	Parameters METParameters `xml:"parameters"`
}

type METParameters struct {
	XMLName    xml.Name `xml:"parameters"`
	Parameters []struct {
		Name  string  `xml:"name,attr"`
		Value float32 `xml:",chardata"`
	} `xml:"parameter"`
	Name string
}
