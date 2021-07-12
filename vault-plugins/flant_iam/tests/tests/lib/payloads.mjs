import Faker from "faker"

export function genFeatureFlag(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        ...override,
    }
}

export function genTenantPayload(override = {}) {
    return {
        identifier: Faker.internet.domainWord(),
        ...override,
    }
}
export function genRoleUpdatePayload(override = {}) {
    const pld = genRoleCreatePayload(override)
    delete pld.name
    return pld
}

export function genRoleCreatePayload(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        description: Faker.lorem.sentence(),
        scope: Math.random() > 0.5 ? "tenant" : "project",
        options_schema: "",
        require_one_of_feature_flags: [],
        ...override,
    }
}

export function genUserPayload(override = {}) {
    return {
        // login: Faker.internet.email(),
        //
        // first_name: Faker.name.firstName(),
        // last_name: Faker.name.lastName(),
        // display_name: Faker.name.lastName(),
        //
        // email: Faker.internet.email(),
        // additional_emails: [],
        //
        // mobile_phone: Faker.phone.phoneNumber(),
        // additional_phones: [],
        identifier: Faker.internet.userName(),
        ...override,
    }
}

export function genMultipassPayload(override = {}) {
    return {
        ttl: Faker.datatype.number(),
        max_ttl: Faker.datatype.number(),
        description: Faker.lorem.sentence(),
        allowed_cidrs: ["10.1.0.0/16"],
        allowed_roles: ["tenant_admin"],
        ...override,
    }
}

export function genPasswordPayload(override = {}) {
    return {
        ttl: Faker.datatype.number(),
        description: Faker.lorem.sentence(),
        allowed_cidrs: ["10.1.0.0/16"],
        allowed_roles: ["tenant_admin"],
        ...override,
    }
}

export function genServiceAccountPayload(override = {}) {
    return {
        identifier: Faker.internet.userName(),
        allowed_cidrs: [],
        token_ttl: Faker.datatype.number(),
        token_max_ttl: Faker.datatype.number(),
        ...override,
    }
}

export function genProjectPayload(override = {}) {
    return {
        identifier: Faker.internet.userName(),
        ...override,
    }
}

export function genGroupPayload(override = {}) {
    return {
        identifier: Faker.lorem.word(),
        subjects: [],
        ...override,
    }
}

export function genRoleBindingPayload(override = {}) {
    return {
        subjects: [],
        ttl: Faker.datatype.number(),
        require_mfa: Math.random() > 0.5,
        ...override,
    }
}

export function genRolePayload(override = {}) {
    return {
        identifier: Faker.lorem.word(),
        subjects: [],
        ...override,
    }
}
