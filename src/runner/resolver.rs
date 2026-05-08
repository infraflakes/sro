use super::context::ExecContext;
use crate::config::ConfigError;
use crate::dsl::ast::Expr;
use std::collections::HashMap;

impl<'a> ExecContext<'a> {
    pub(super) fn resolve_expr(&self, expr: &Expr) -> Result<String, ConfigError> {
        match expr {
            Expr::BacktickLit { parts, .. } => {
                let mut result = String::new();
                for part in parts {
                    if part.is_var {
                        let var_name = part.value.trim_start_matches('$');
                        if let Some(value) = self.vars.get(var_name) {
                            result.push_str(value);
                        } else {
                            return Err(ConfigError::Validation(format!(
                                "undefined variable: ${}",
                                var_name
                            )));
                        }
                    } else {
                        result.push_str(&part.value);
                    }
                }
                Ok(result)
            }
            Expr::VarRef { name, .. } => {
                if let Some(value) = self.vars.get(name) {
                    Ok(value.clone())
                } else {
                    Err(ConfigError::Validation(format!(
                        "undefined variable: ${}",
                        name
                    )))
                }
            }
        }
    }

    pub(super) fn build_env(&self) -> Vec<(String, String)> {
        let mut env: HashMap<String, String> = HashMap::new();

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
