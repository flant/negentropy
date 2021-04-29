import { API, stringifyQuery } from "./api.mjs"
import Faker from "faker"
import { join } from "path"

export class TenantEndpointBuilder {
    one(p = {}, q = {}) {
        return join("/tenant", p.tenant) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return "/tenant" + stringifyQuery(q)
    }


    privileged(p = {}, q = {}) {
        return join("/tenant", "privileged") + stringifyQuery(q)
    }
}

export class TenantAPI {
    constructor(client) {
        this.api = new API(client, new TenantEndpointBuilder())
    }

    create(payload, opts) {
        return this.api.create({ payload, opts })
    }

    createPriveleged(payload, opts) {
        return this.api.createPrivileged({ payload, opts })
    }

    read(id, opts) {
        const params = { tenant: id }
        return this.api.read({ params, opts })
    }

    update(id, payload, opts) {
        const params = { tenant: id }
        return this.api.update({ params, payload, opts })
    }

    delete(id, opts) {
        const params = { tenant: id }
        return this.api.delete({ params, opts })
    }

    list(opts) {
        const query = { list: true }
        return this.api.list({ query, opts })
    }
}

export function genTenantPayload(override = {}) {
    return {
        identifier: Faker.internet.domainWord(),
        ...override,
    }
}
