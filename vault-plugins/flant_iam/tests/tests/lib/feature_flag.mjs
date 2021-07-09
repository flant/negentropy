import Faker from "faker"

export function genFeatureFlag(override = {}) {
    return {
        name: Faker.internet.domainWord(),
        ...override,
    }
}
