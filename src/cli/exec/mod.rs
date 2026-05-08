pub mod par;
pub mod seq;

pub use par::run as run_par;
pub use seq::run as run_seq;
