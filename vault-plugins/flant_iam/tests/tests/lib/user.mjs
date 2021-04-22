import { stringifyQuery } from "./api.mjs"
import Faker from "faker"
import { TenantEndpointBuilder } from "./tenant.mjs"
import { join } from "path"

export class UserEndpointBuilder extends TenantEndpointBuilder {
    one(p = {}, q = {}) {
        return join(super.one(p), "user", p.user) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return join(super.one(p), "user") + stringifyQuery(q)
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
