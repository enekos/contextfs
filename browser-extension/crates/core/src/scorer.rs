// mairu-ext/crates/core/src/scorer.rs
use rust_stemmers::{Algorithm, Stemmer};
use std::collections::HashMap;
use std::sync::LazyLock;

/// Porter2 (Snowball English) stemmer.
static STEMMER: LazyLock<Stemmer> = LazyLock::new(|| Stemmer::create(Algorithm::English));

/// Synonym map: stemmed term → list of stemmed synonyms.
/// Source data is embedded from `synonyms.json` at compile time;
/// all keys and values are stemmed on first use.
static SYNONYMS: LazyLock<HashMap<String, Vec<String>>> = LazyLock::new(|| {
    let raw: HashMap<String, Vec<String>> =
        serde_json::from_str(include_str!("../synonyms.json")).unwrap_or_default();
    raw.into_iter()
        .map(|(k, v)| {
            let stemmed_key = stem(&k);
            let stemmed_vals: Vec<String> = v.iter().map(|w| stem(w)).collect();
            (stemmed_key, stemmed_vals)
        })
        .collect()
});

fn stem(word: &str) -> String {
    STEMMER.stem(&word.to_lowercase()).to_string()
}

fn synonyms(stemmed_term: &str) -> &[String] {
    SYNONYMS
        .get(stemmed_term)
        .map(|v| v.as_slice())
        .unwrap_or(&[])
}

fn tokenize(text: &str) -> Vec<String> {
    text.to_lowercase()
        .split(|c: char| !c.is_alphanumeric())
        .filter(|w| w.len() > 1)
        .map(stem)
        .collect()
}

/// A lightweight TF-IDF index with Porter2 stemming and synonym expansion.
pub struct TfIdfIndex {
    /// doc_id -> term -> count
    docs: HashMap<String, HashMap<String, usize>>,
    /// term -> number of docs containing it
    doc_freq: HashMap<String, usize>,
    total_docs: usize,
}

impl Default for TfIdfIndex {
    fn default() -> Self {
        Self::new()
    }
}

impl TfIdfIndex {
    pub fn new() -> Self {
        Self {
            docs: HashMap::new(),
            doc_freq: HashMap::new(),
            total_docs: 0,
        }
    }

    pub fn add(&mut self, doc_id: &str, text: &str) {
        self.remove(doc_id);

        let tokens = tokenize(text);
        let mut term_counts: HashMap<String, usize> = HashMap::new();
        for token in &tokens {
            *term_counts.entry(token.clone()).or_insert(0) += 1;
        }
        for term in term_counts.keys() {
            *self.doc_freq.entry(term.clone()).or_insert(0) += 1;
        }
        self.docs.insert(doc_id.to_string(), term_counts);
        self.total_docs += 1;
    }

    pub fn remove(&mut self, doc_id: &str) {
        if let Some(term_counts) = self.docs.remove(doc_id) {
            for term in term_counts.keys() {
                if let Some(df) = self.doc_freq.get_mut(term) {
                    *df = df.saturating_sub(1);
                    if *df == 0 {
                        self.doc_freq.remove(term);
                    }
                }
            }
            self.total_docs = self.total_docs.saturating_sub(1);
        }
    }

    pub fn search(&self, query: &str) -> Vec<(String, f64)> {
        if self.total_docs == 0 {
            return Vec::new();
        }
        let raw_tokens = tokenize(query);
        if raw_tokens.is_empty() {
            return Vec::new();
        }

        // Expand with synonyms (values are already stemmed)
        let mut query_tokens = raw_tokens.clone();
        for token in &raw_tokens {
            for syn in synonyms(token) {
                if !query_tokens.contains(syn) {
                    query_tokens.push(syn.clone());
                }
            }
        }

        let mut scores: Vec<(String, f64)> = self
            .docs
            .iter()
            .map(|(doc_id, term_counts)| {
                let total_terms: usize = term_counts.values().sum();
                let score: f64 = query_tokens
                    .iter()
                    .map(|qt| {
                        let tf = *term_counts.get(qt).unwrap_or(&0) as f64 / total_terms as f64;
                        let df = *self.doc_freq.get(qt).unwrap_or(&0) as f64;
                        let idf = if df > 0.0 {
                            (self.total_docs as f64 / df).ln() + 1.0
                        } else {
                            0.0
                        };
                        tf * idf
                    })
                    .sum();
                (doc_id.clone(), score)
            })
            .filter(|(_, score)| *score > 0.0)
            .collect();

        scores.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
        scores
    }

    /// Search with importance-weighted re-ranking.
    pub fn search_with_importance<F>(&self, query: &str, importance_fn: F) -> Vec<(String, f64)>
    where
        F: Fn(&str) -> f64,
    {
        let mut results = self.search(query);
        for (doc_id, score) in &mut results {
            let importance = importance_fn(doc_id);
            *score *= 1.0 + importance * 0.01;
        }
        results.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
        results
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_empty_index_returns_empty() {
        let index = TfIdfIndex::new();
        let results = index.search("anything");
        assert!(results.is_empty());
    }

    #[test]
    fn test_exact_match_ranks_first() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "rust programming language");
        index.add("doc2", "python programming language");
        index.add("doc3", "rust metal oxidation");
        let results = index.search("rust programming");
        assert_eq!(results[0].0, "doc1");
    }

    #[test]
    fn test_idf_boosts_rare_terms() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "the common word unique_xyz");
        index.add("doc2", "the common word something");
        index.add("doc3", "the common word another");
        let results = index.search("unique_xyz");
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].0, "doc1");
    }

    #[test]
    fn test_scores_are_descending() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "rust rust rust");
        index.add("doc2", "rust python");
        let results = index.search("rust");
        assert!(results.len() == 2);
        assert!(results[0].1 >= results[1].1);
    }

    #[test]
    fn test_stem_porter2() {
        // Verify Porter2 produces the expected stems for our key search words
        assert_eq!(stem("running"), "run");
        assert_eq!(stem("stopping"), "stop");
        assert_eq!(stem("testing"), "test");
        assert_eq!(stem("configuration"), "configur");
        assert_eq!(stem("configured"), "configur");
        assert_eq!(stem("authentication"), "authent");
        assert_eq!(stem("authenticate"), "authent");
    }

    #[test]
    fn test_stemming_matches_in_search() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "The application is running smoothly");
        index.add("doc2", "Python is a scripting language");
        let results = index.search("run");
        assert!(!results.is_empty(), "'run' should match 'running'");
        assert_eq!(results[0].0, "doc1");
    }

    #[test]
    fn test_synonym_expansion() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "Authentication failed with invalid credentials");
        index.add("doc2", "The database connection is slow");
        // "login" is a synonym for "authenticate" (same Porter2 stem: "authent")
        let results = index.search("login");
        assert!(
            !results.is_empty(),
            "'login' should match via synonym for 'authentication'"
        );
        assert_eq!(results[0].0, "doc1");
    }

    #[test]
    fn test_synonyms_are_data_driven() {
        // Verify the synonym map loaded from synonyms.json
        assert!(
            !synonyms("authent").is_empty(),
            "authent should have synonyms (login/auth)"
        );
        assert!(
            synonyms("function").contains(&"method".to_string()),
            "function should expand to method"
        );
        assert!(
            synonyms("method").contains(&"function".to_string()),
            "method should expand to function"
        );
    }

    #[test]
    fn test_remove_doc() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "hello world");
        index.add("doc2", "hello earth");
        assert_eq!(index.total_docs, 2);

        index.remove("doc1");
        assert_eq!(index.total_docs, 1);

        let results = index.search("world");
        assert!(results.is_empty());

        let results = index.search("hello");
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].0, "doc2");
    }

    #[test]
    fn test_re_add_replaces() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "old content here");
        index.add("doc1", "new content now");
        assert_eq!(index.total_docs, 1);

        let results = index.search("old");
        assert!(results.is_empty());

        let results = index.search("new");
        assert_eq!(results.len(), 1);
    }

    #[test]
    fn test_search_with_importance() {
        let mut index = TfIdfIndex::new();
        index.add("doc1", "rust programming guide");
        index.add("doc2", "rust programming tutorial");

        let results = index.search_with_importance("rust programming", |doc_id| {
            if doc_id == "doc2" {
                100.0
            } else {
                0.0
            }
        });
        assert_eq!(results[0].0, "doc2");
    }
}
