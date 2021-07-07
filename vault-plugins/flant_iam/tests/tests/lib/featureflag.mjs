import { API, stringifyQuery } from "./api.mjs"
import Faker from "faker"
import { join } from "path"

export class FeatureFlagEndpointBuilder {
    one(p = {}, q = {}) {
        return join("/featureflag", p.name) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return "/featureflag" + stringifyQuery(q)
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
