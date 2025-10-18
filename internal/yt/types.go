package yt

import (
	"encoding/json"
	"fmt"
)

type ytdlpChapter struct {
	StartTime float64 `json:"start_time"` // champ moderne, à préférer
	Start     float64 `json:"start"`      // fallback
	Title     string  `json:"title"`
}
type subtitleItem struct {
	Ext string `json:"ext"`
	URL string `json:"url"`
}

// ytdlpOutput représente la sortie JSON brute retournée par yt-dlp pour une vidéo.
// Chaque champ correspond à un élément présent dans le JSON de yt-dlp.
//
// Subtitles et AutomaticCaptions sont des maps où :
//   - la clé (string) correspond au code langue de la piste (ex. "fr", "en", "fr-orig").
//   - la valeur ([]subtitleItem) est un slice listant toutes les pistes disponibles pour cette langue,
//     chaque élément contenant au minimum l'extension du fichier (Ext) et l'URL pour le télécharger.
type ytdlpOutput struct {
	ID                string                    `json:"id"`
	Title             string                    `json:"title"`
	Uploader          string                    `json:"uploader"`
	UploadDate        string                    `json:"upload_date"`
	Timestamp         int64                     `json:"timestamp"` // en Unix epoch
	Categories        []string                  `json:"categories"`
	YtTags            []string                  `json:"tags"`
	Description       string                    `json:"description"`
	Chapters          []ytdlpChapter            `json:"chapters"`
	Subtitles         map[string][]subtitleItem `json:"subtitles"`
	AutomaticCaptions map[string][]subtitleItem `json:"automatic_captions"`
}

// ExtractedRaw contient le JSON raw, une liste de lignes d'avertissements
type ExtractedRaw struct {
	JSON     []byte
	Warnings []string
}

// PrettyJSON retourne un json indenté
func (r *ExtractedRaw) PrettyJSON() ([]byte, error) {
	var obj any
	if err := json.Unmarshal(r.JSON, &obj); err != nil {
		return nil, err
	}
	return json.MarshalIndent(obj, "", "  ")
}

// PrintWarning affiche les avertissements de yt-dlp
func (v *ExtractedRaw) PrintWarnings() {
	if len(v.Warnings) == 0 {
		return
	}
	fmt.Println("⚠️  Avertissements yt-dlp :")
	for _, w := range v.Warnings {
		fmt.Printf("  - %s\n", w)
	}
}

// YtDlp représente la commande yt-dlp à exécuter (nom de binaire ou chemin) + args.
type YtDlp struct {
	Name   string
	Path   string // chemin vers l'exe
	Config YtDlpConfig
}

func (y YtDlp) DisplayInfo() {
	fmt.Printf("Name: %s\n", y.Name)
	fmt.Printf("Path: %s\n", y.Path)
}

func (y YtDlp) ShowName() {
	fmt.Println("Name:", y.Name)
}

func (y YtDlp) ShowPath() {
	fmt.Println("yt-dlp path:", y.Path)
}
