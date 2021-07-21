use anyhow::Result;
use libnss::interop::Response;
use libnss::passwd::{Passwd, PasswdHooks};
use rusqlite::OpenFlags;
use rusqlite::{params, Connection, Row, NO_PARAMS};

use crate::db::from_result;
use crate::db::DB_PATH;

pub struct SqlitePasswd;
libnss_passwd_hooks!(flantauth, SqlitePasswd);

impl PasswdHooks for SqlitePasswd {
    fn get_all_entries() -> Response<Vec<Passwd>> {
        let entries =
            Connection::open_with_flags(&DB_PATH as &str, OpenFlags::SQLITE_OPEN_READ_ONLY)
                .map_err(Into::into)
                .and_then(get_all_entries);

        from_result(entries)
    }

    fn get_entry_by_uid(uid: libc::uid_t) -> Response<Passwd> {
        let entry = Connection::open_with_flags(&DB_PATH as &str, OpenFlags::SQLITE_OPEN_READ_ONLY)
            .map_err(Into::into)
            .and_then(|conn| get_entry_by_uid(conn, uid));

        from_result(entry)
    }

    fn get_entry_by_name(name: String) -> Response<Passwd> {
        let entry = Connection::open_with_flags(&DB_PATH as &str, OpenFlags::SQLITE_OPEN_READ_ONLY)
            .map_err(Into::into)
            .and_then(|conn| get_entry_by_name(conn, &name));

        from_result(entry)
    }
}

fn get_all_entries(conn: Connection) -> Result<Vec<Passwd>> {
    conn.prepare("SELECT name, uid, gid, gecos, homedir, shell, hashed_pass FROM users")?
        .query_and_then(NO_PARAMS, from_row)?
        .collect()
}
fn get_entry_by_uid(conn: Connection, uid: u32) -> Result<Passwd> {
    conn.query_row_and_then(
        "SELECT name, uid, gid, gecos, homedir, shell, hashed_pass FROM users WHERE uid = ?1",
        params![uid],
        from_row,
    )
}
fn get_entry_by_name(conn: Connection, name: &str) -> Result<Passwd> {
    conn.query_row_and_then(
        "SELECT name, uid, gid, gecos, homedir, shell, hashed_pass FROM users WHERE name = ?1",
        params![name],
        from_row,
    )
}

fn from_row(row: &Row) -> Result<Passwd> {
    Ok(Passwd {
        name: row.get(0)?,
        uid: row.get(1)?,
        gid: row.get(2)?,
        gecos: row.get(3)?,
        dir: row.get(4)?,
        shell: row.get(5)?,
        passwd: "x".to_string(),
    })
}