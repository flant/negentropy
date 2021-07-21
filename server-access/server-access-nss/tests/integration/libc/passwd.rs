use std::sync;

use crate::integration::common;
use crate::integration::libc::nss;

fn setup() {
    nss::setup().expect("failed to setup nss");

    static INIT: sync::Once = sync::Once::new();
    common::setup(|conn| {
        INIT.call_once(|| {
            conn.execute_batch(
                r#"
                INSERT INTO users VALUES ("test-user", 123, 321, "some comment", "/home/test-user", "/fake-shell", "*");
                INSERT INTO users VALUES ("second-user", 234, 321, "some comment", "/home/second", "/fake-shell", "*");
                "#,
            ).expect("failed to create test users");
        });
    })
}

#[test]
fn get_all_users() -> common::TestResult<()> {
    setup();

    let user_list = unsafe { users::all_users() };
    let user_count = user_list.count();

    assert_eq!(2, user_count);

    Ok(())
}

#[test]
fn get_existing_user() -> common::TestResult<()> {
    setup();

    let user_by_name =
        users::get_user_by_name("test-user").expect("failed to find expected user by name");
    let user_by_id = users::get_user_by_uid(123).expect("failed to find expected user by uid");

    assert_eq!("test-user", user_by_name.name());
    assert_eq!(123, user_by_name.uid());

    assert_eq!(user_by_name.name(), user_by_id.name());
    assert_eq!(user_by_name.uid(), user_by_id.uid());

    Ok(())
}

#[test]
fn get_missing_user() -> common::TestResult<()> {
    setup();

    let user_by_name = users::get_user_by_name("missing-user");
    let user_by_id = users::get_user_by_uid(999);

    assert!(user_by_name.is_none());
    assert!(user_by_id.is_none());

    Ok(())
}