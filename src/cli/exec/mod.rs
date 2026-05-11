pub mod par;
pub mod seq;

mod func;

pub use func::run as run_fn;
pub use par::run as run_par;
pub use seq::run as run_seq;
