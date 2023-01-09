package analysis

// type PublicationSet struct {
// 	Content     string
// 	TruthValues []bool
// }

// // A truthValueProbability of 1 means that all publications are true.
// func generatePublicationsSet(nbr int, content string, truthValueProbability float32) PublicationSet {
// 	truthValues := make([]bool, nbr)
// 	for i := 0; i < nbr; i++ {
// 		r := rand.Float32()
// 		if r <= truthValueProbability {
// 			truthValues[i] = true
// 		} else {
// 			truthValues[i] = false
// 		}
// 	}
// 	return PublicationSet{
// 		Content:     content,
// 		TruthValues: truthValues,
// 	}
// }

// type ClusterPublicationSet struct {
// 	nodesPublications map[string]PublicationSet
// }
