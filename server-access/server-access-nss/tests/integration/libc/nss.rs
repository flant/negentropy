use std::env;
use std::ffi::CString;
use std::io;
use std::os;
use std::sync;

// We can really only mutuate libc's process memory _once_ per process. So we
// need to enforce that.
static NSS_INIT: sync::Once = sync::Once::new();
pub fn setup() -> io::Result<()> {
    // Make sure we have a LD_LIBRARY_PATH that'll probably contain
    // libnss_sqlite.so on it to begin with.
    sanity_check_env();

    // Setup symlink so our build artifact is named correctly
    os::unix::fs::symlink("libnss_flantauth.so", "target/debug/libnss_flantauth.so.2").or_else(
        |e: io::Error| match e.kind() {
            // If we try to make the symlink and it alraedy exists, no problem
            io::ErrorKind::AlreadyExists => Ok(()),
            // else pass through
            _ => Err(e),
        },
    )?;

    NSS_INIT.call_once(|| {
        for db in &["passwd"] {
            nss_configure_lookup(db, "flantauth")
        }
    });

    Ok(())
}

fn sanity_check_env() {
    let ld_path = env::var("LD_LIBRARY_PATH")
        .expect("LD_LIBRARY_PATH isn't set. This is extremely unlikely to work (we won't be able to find the libnss_sqlite.so.2 to load). This should've been taken care of by `cargo test` for you.");

    assert!(
        ld_path.contains("target/debug:") || ld_path.ends_with("target/debug"),
        "LD_LIBRARY_PATH doesn't seem to contain `$crate/target/debug` in its path list.\nLD_LIBRARY_PATH={}", ld_path
    );
}

extern "C" {
    fn __nss_configure_lookup(db: *const libc::c_char, config: *const libc::c_char);
}

fn nss_configure_lookup(db: &str, nss_config: &str) {
    let db = CString::new(db).unwrap();
    let nss_config = CString::new(nss_config).unwrap();

    unsafe { __nss_configure_lookup(db.as_ptr(), nss_config.as_ptr()) }
}