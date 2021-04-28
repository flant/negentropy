import { stringifyQuery } from "./api.mjs"
import Faker from "faker"
import { TenantEndpointBuilder } from "./tenant.mjs"
import { join } from "path"

export class SubTenantEntrypointBuilder extends TenantEndpointBuilder {
    constructor(name) {
        super()
        this.entryName = name // e.g. "user" or "service_account"
    }

    one(p = {}, q = {}) {
        return join(super.one(p), this.entryName, p[this.entryName]) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return join(super.one(p), this.entryName) + stringifyQuery(q)
    }

    privileged(p = {}, q = {}) {
        return join(super.one(p), this.entryName, "privileged") + stringifyQuery(q)
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
        users: [],
        groups: [],
        service_accounts: [],
        ...override,
    }
}

export function genRoleBindingPayload(override = {}) {
    return {
        users: [],
        groups: [],
        service_accounts: [],
        ttl: Faker.datatype.number(),
        require_mfa: Math.random() > 0.5,
        ...override,
    }
}


export function genRolePayload(override = {}) {
    return {
        identifier: Faker.lorem.word(),
        // users,
        // groups,
        // serviceAccounts
        ...override,
    }
}
