use std::sync;

use crate::integration::common;
use crate::integration::libc::nss;

fn setup() {
    // nss::setup().expect("failed to hook libnss");
    //
    // static INIT: sync::Once = sync::Once::new();
    // common::setup(|conn| {
    //     INIT.call_once(|| {
    //         conn.execute_batch(
    //             r#"
    //             INSERT INTO groups VALUES ("test-group", 321);
    //             "#,
    //         )
    //         .expect("failed to create test groups")
    //     })
    // })
}

#[test]
fn get_existing_group() -> common::TestResult<()> {
    setup();

    let group_by_name =
        users::get_group_by_name("test-group").expect("failed to find group by name");
    let group_by_id = users::get_group_by_gid(321).expect("failed to find group by id");

    assert_eq!("test-group", group_by_name.name());
    assert_eq!(321, group_by_name.gid());

    assert_eq!(group_by_name.name(), group_by_id.name());
    assert_eq!(group_by_name.gid(), group_by_id.gid());

    Ok(())
}

#[test]
fn get_missing_group() -> common::TestResult<()> {
    setup();

    let user_by_name = users::get_group_by_name("missing-group");
    let user_by_id = users::get_group_by_gid(999);

    assert!(user_by_name.is_none());
    assert!(user_by_id.is_none());

    Ok(())
}