#[macro_use]
extern crate lazy_static;
#[macro_use]
extern crate libnss;

mod db;

pub mod group;
pub mod passwd;
pub mod shadow;