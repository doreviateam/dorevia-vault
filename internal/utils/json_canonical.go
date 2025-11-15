package utils

import (
	"encoding/json"
	"sort"
)

// CanonicalizeJSON canonicalise un JSON pour garantir un hash stable
// Algorithme :
// 1. Parser le JSON en map[string]interface{}
// 2. Trier récursivement les clés
// 3. Supprimer les champs null
// 4. Normaliser les nombres (10.0 → 10 si entier)
// 5. Marshal avec json.Marshal (pas d'indentation, pas d'espaces)
func CanonicalizeJSON(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	// Normaliser récursivement
	normalized := normalizeValue(obj)

	// Marshal sans indentation (compact)
	return json.Marshal(normalized)
}

// normalizeValue normalise récursivement une valeur
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Créer une map triée
		sorted := make(map[string]interface{})
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Copier les valeurs normalisées dans l'ordre trié
		for _, k := range keys {
			// Ignorer les valeurs null
			if val[k] != nil {
				sorted[k] = normalizeValue(val[k])
			}
		}
		return sorted

	case []interface{}:
		// Normaliser chaque élément du tableau
		normalized := make([]interface{}, len(val))
		for i, item := range val {
			normalized[i] = normalizeValue(item)
		}
		return normalized

	case float64:
		// Normaliser les nombres (10.0 → 10 si entier)
		if val == float64(int64(val)) {
			return int64(val)
		}
		return val

	default:
		return val
	}
}

