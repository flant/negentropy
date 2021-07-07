import { API, stringifyQuery } from "./api.mjs"
import Faker from "faker"
import { join } from "path"

export class RoleEndpointBuilder {
    one(p = {}, q = {}) {
        return join("/role", p.role) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return "/role" + stringifyQuery(q)
    }
}

export class RoleAPI {
    constructor(client) {
        this.api = new API(client, new RoleEndpointBuilder())
    }

    create(payload, opts) {
        return this.api.create({ payload, opts })
    }

    createPriveleged(payload, opts) {
        return this.api.createPrivileged({ payload, opts })
    }

    read(id, opts) {
        const params = { role: id }
        return this.api.read({ params, opts })
    }

    update(id, payload, opts) {
        const params = { role: id }
        return this.api.update({ params, payload, opts })
    }

    delete(id, opts) {
        const params = { role: id }
        return this.api.delete({ params, opts })
    }

    list(opts) {
        const query = { list: true }
        return this.api.list({ query, opts })
    }
}

export function genRoleUpdatePayload(override = {}) {
    return {
        description: Faker.lorem.sentence(),
        type: Math.random() > 0.5 ? "tenant" : "project",
        options_schema: "",
        require_one_of_featureflags: [],
        ...override,
    }
}

export function genRoleCreatePayload(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        description: Faker.lorem.sentence(),
        type: Math.random() > 0.5 ? "tenant" : "project",
        options_schema: "",
        require_one_of_featureflags: [],
        ...override,
    }
}
