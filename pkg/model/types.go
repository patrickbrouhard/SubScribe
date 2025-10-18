package model

import "fmt"

// Seconds est un alias explicite pour représenter une durée en secondes.
type Seconds int64

// TimestampHHMMSS formate Seconds en "HH:MM:SS" (toujours 2 chiffres par composant).
// Exemple : 65 -> "00:01:05", 3661 -> "01:01:01".
func (s Seconds) TimestampHHMMSS() string {
	total := int64(s)
	h := total / 3600
	m := (total % 3600) / 60
	sec := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
}

func (s Seconds) Milliseconds() int64 {
	return int64(s) * 1000
}

// constantes pour les formats de fichiers
type Format string

const (
	FormatTXT      Format = "txt"
	FormatMARKDOWN Format = "md"
	FormatJSON3    Format = "json3"
	FormatSRT      Format = "srt"
	FormatVTT      Format = "vtt"
)

// du format en chaine à la constante de type Format, return une erreur si format inconnu
func ParseFormat(s string) (Format, error) {
	switch s {
	case "txt":
		return FormatTXT, nil
	case "md":
		return FormatMARKDOWN, nil
	case "json3":
		return FormatJSON3, nil
	case "srt":
		return FormatSRT, nil
	case "vtt":
		return FormatVTT, nil
	default:
		return "", fmt.Errorf("format demandé inconnu: %s", s)
	}
}

func (f Format) IsSubtitle() bool {
	return f == FormatJSON3 || f == FormatSRT || f == FormatVTT
}

func (f Format) IsTextual() bool {
	return f == FormatTXT || f == FormatMARKDOWN
}

func (f Format) Extension() string {
	return "." + string(f)
}

func (f Format) String() string {
	return string(f)
}
