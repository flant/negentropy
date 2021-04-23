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
}


export function genUserPayload(override = {}) {
    return {
        login: Faker.internet.email(),

        first_name: Faker.name.firstName(),
        last_name: Faker.name.lastName(),
        display_name: Faker.name.lastName(),

        email: Faker.internet.email(),
        additional_emails: [],

        mobile_phone: Faker.phone.phoneNumber(),
        additional_phones: [],

        ...override,
    }
}


export function genServiceAccountPayload(override = {}) {
    return {
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
        // users,
        // groups,
        // serviceAccounts
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
