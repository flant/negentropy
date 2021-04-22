import { API } from "./api.mjs"
import Faker from "faker"

export class TenantEndpointBuilder {
    create(p = {}, q = {}) {
        return "/tenant"
    }

    read(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    update(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    delete(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    list(p = {}, q = {}) {
        return "/tenant?list=true"
    }
}

export class TenantAPI {
    constructor(client) {
        this.api = new API(client, new TenantEndpointBuilder())
    }

    create(payload, opts) {
        return this.api.create({ payload, opts })
    }

    read(id, opts) {
        return this.api.read({ params: { tenant: id }, opts })
    }

    update(id, payload, opts) {
        return this.api.update({ params: { tenant: id }, payload, opts })
    }

    delete(id, opts) {
        return this.api.delete({ params: { tenant: id }, opts })
    }

    list(opts) {
        return this.api.list({ opts })
    }
}

export function genTenantPayload(override = {}) {
    return {
        name: Faker.lorem.word(),
        ...override,
    }
}
