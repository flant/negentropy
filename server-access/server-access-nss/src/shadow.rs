use anyhow::Result;
use libnss::interop::Response;
use libnss::shadow::{Shadow, ShadowHooks};
use rusqlite::OpenFlags;
use rusqlite::{params, Connection, Row, NO_PARAMS};

use crate::db::from_result;
use crate::db::DB_PATH;

pub struct SqliteShadow;
libnss_shadow_hooks!(flantauth, SqliteShadow);

impl ShadowHooks for SqliteShadow {
    fn get_all_entries() -> Response<Vec<Shadow>> {
        let entries =
            Connection::open_with_flags(&DB_PATH as &str, OpenFlags::SQLITE_OPEN_READ_ONLY)
                .map_err(Into::into)
                .and_then(get_all_entries);

        from_result(entries)
    }

    fn get_entry_by_name(name: String) -> Response<Shadow> {
        let entry = Connection::open_with_flags(&DB_PATH as &str, OpenFlags::SQLITE_OPEN_READ_ONLY)
            .map_err(Into::into)
            .and_then(|conn| get_entry_by_name(conn, &name));

        from_result(entry)
    }
}

fn get_all_entries(conn: Connection) -> Result<Vec<Shadow>> {
    conn.prepare("SELECT name, hashed_pass FROM users")?
        .query_and_then(NO_PARAMS, from_row)?
        .collect()
}
fn get_entry_by_name(conn: Connection, name: &str) -> Result<Shadow> {
    conn.query_row_and_then(
        "SELECT name, hashed_pass FROM users WHERE name = ?1",
        params![name],
        from_row,
    )
}

fn from_row(row: &Row) -> Result<Shadow> {
    Ok(Shadow {
        name: row.get(0)?,
        passwd: row.get(1)?,
        last_change: -1,
        change_min_days: -1,
        change_max_days: -1,
        change_warn_days: -1,
        change_inactive_days: -1,
        expire_date: -1,
        reserved: (!0u32) as u64,
    })
}