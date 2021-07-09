import Faker from "faker"
import { join } from "path"
import { API } from "./api.mjs"
import { stringifyQuery } from "./endpoint_builder.mjs"
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

    async create(payload, opts) {
        const { data: body } = await this.api.create({ payload, opts })
        if (body.data) return body.data.tenant
        return body
    }

    async createPriveleged(payload, opts) {
        const { data: body } = await this.api.createPrivileged({
            payload,
            opts,
        })
        if (body.data) return body.data.tenant
        return body
    }

    async read(id, opts) {
        const params = { tenant: id }
        const { data: body } = await this.api.read({ params, opts })
        if (body.data) return body.data.tenant
        return body
    }

    async update(id, payload, opts) {
        const params = { tenant: id }
        const { data: body } = await this.api.update({ params, payload, opts })
        if (body.data) return body.data.tenant
        return body
    }

    async delete(id, opts) {
        const params = { tenant: id }
        const { data: body } = await this.api.delete({ params, opts })
        if (body.data) return body.data.tenant
        return body
    }

    async list(opts) {
        const query = { list: true }
        const { data: body } = await this.api.list({ query, opts })
        if (body.data) return body.data.uuids
        return body
    }
}

export function genTenantPayload(override = {}) {
    return {
        identifier: Faker.internet.domainWord(),
        ...override,
    }
}
