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

    create(payload, opts) {
        return this.api.create({ payload, opts })
    }

    delete(name, opts) {
        const params = { name }
        return this.api.delete({ params, opts })
    }

    list(opts) {
        const query = { list: true }
        return this.api.list({ query, opts })
    }
}

export function genFeatureFlag(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        ...override,
    }
}
