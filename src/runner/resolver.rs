use super::context::ExecContext;
use crate::config::ConfigError;
use crate::dsl::ast::Expr;

impl<'a> ExecContext<'a> {
    pub(super) fn resolve_expr(&self, expr: &Expr) -> Result<String, ConfigError> {
        expr.resolve(&self.vars).map_err(ConfigError::Validation)
    }

    pub(super) fn build_env(&self) -> Vec<(String, String)> {
        let mut env: std::collections::HashMap<String, String> = std::collections::HashMap::new();

        for (key, value) in std::env::vars() {
            env.insert(key, value);
        }

        for layer in &self.env_stack {
            for (key, value) in layer {
                env.insert(key.clone(), value.clone());
            }
        }

        env.into_iter().collect()
    }
}
