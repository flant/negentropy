use std::env;
use std::path::Path;

use lazy_static::lazy_static;
use tempfile::{NamedTempFile, TempPath};

pub type TestResult<T> = Result<T, Box<dyn std::error::Error>>;

lazy_static! {
    // Note: This leaks a file every run, I don't know a nice way to fix this.
    // We have to only have a single tempfile for the whole run of the process
    // given our current impl.
    pub static ref DB_PATH: TempPath = NamedTempFile::new()
        .expect("failed to setup tempfile for database")
        .into_temp_path();
}

pub fn setup<F: FnOnce(rusqlite::Connection)>(user_setup: F) {
    let conn = setup_db(&DB_PATH).expect("failed to create test database and set env vars");
    user_setup(conn);
}

// If you don't run this first, lazy_static defined PATHs may get set
// incorrectly
fn setup_db(path: &Path) -> Result<rusqlite::Connection, Box<dyn std::error::Error>> {
    let conn = rusqlite::Connection::open(path)?;

    conn.execute_batch(include_str!("./testdata/db.sql"))?;

    env::set_var("NSS_FLANTAUTH_PASSWD_PATH", path);

    Ok(conn)
}