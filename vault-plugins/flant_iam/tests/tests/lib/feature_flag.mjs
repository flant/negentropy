import Faker from "faker"
import { join } from "path"
import { API } from "./api.mjs"
import { stringifyQuery } from "./endpoint_builder.mjs"
export class FeatureFlagEndpointBuilder {
    one(p = {}, q = {}) {
        return join("/feature_flag", p.name) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return "/feature_flag" + stringifyQuery(q)
    }
}

export class FeatureFlagAPI {
    constructor(client) {
        this.api = new API(client, new FeatureFlagEndpointBuilder())
    }

    async create(payload, opts) {
        const { data: body } = await this.api.create({ payload, opts })
        if (body.data) return body.data.feature_flag
        return body
    }

    async delete(name, opts) {
        const params = { name }
        const { data: body } = await this.api.delete({ params, opts })
        if (body.data) return body.data.feature_flag
        return body
    }

    async list(opts) {
        const query = { list: true }
        const { data: body } = await this.api.list({ query, opts })
        if (body.data) return body.data.names
        return body
    }
}

export function genFeatureFlag(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        ...override,
    }
}
