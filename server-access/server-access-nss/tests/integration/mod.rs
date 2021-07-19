mod common;

mod interface {
    mod shadow;
}

// #[cfg(target_os = "linux")]
mod libc {
    // Support tooling
    mod nss;

    // Tests
    mod groups;
    mod passwd;
}