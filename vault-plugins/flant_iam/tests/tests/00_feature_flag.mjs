import { expect } from "chai"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { FeatureFlagAPI, genFeatureFlag } from "./lib/feature_flag.mjs"

describe("Feature flag", function () {
    const rootClient = getClient(rootToken)
    const root = new FeatureFlagAPI(rootClient)

    describe("payload", () => {
        describe("name", () => {
            after("clean", async () => {
                const names = await root.list()
                const deletions = names.map((name) => root.delete(name))
                await Promise.all(deletions)
            })

            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genFeatureFlag({
                        name: Math.round(Math.random() * 1e9),
                    }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent name forbidden",
                    payload: (() => {
                        const p = genFeatureFlag({})
                        delete p.name
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genFeatureFlag({ name: "" }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "array forbidden",
                    payload: genFeatureFlag({ name: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genFeatureFlag({ name: { a: 1 } }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
            ]

            invalidCases.forEach((x) =>
                it(x.title, async () => {
                    await root.create(x.payload, {
                        validateStatus: x.validateStatus,
                    })
                }),
            )
        })
    })

    it("can be created", async () => {
        const payload = genFeatureFlag()

        const ff = await root.create(payload)

        expect(ff.name).to.eq(payload.name)
    })

    it("can be listed", async () => {
        const payload = genFeatureFlag()
        await root.create(payload)

        const names = await root.list()

        expect(names).to.be.an("array")
    })

    it("has identifying fields in list", async () => {
        const payload = genFeatureFlag()
        const ff = await root.create(payload)

        const list = await root.list()

        expect(list).to.include(ff.name)
    })

    it("can be deleted", async () => {
        const createPld = genFeatureFlag()

        const ff = await root.create(createPld)

        await root.delete(ff.name)

        const list = await root.list()
        expect(list).to.not.include(ff.name)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)

        it("cannot delete, gets 404", async () => {
            await root.delete("no-such", opts)
        })
    })

    describe("no access", function () {
        describe("when unauthenticated", function () {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function () {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const unauth = new FeatureFlagAPI(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create(genFeatureFlag(), opts)
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list(opts)
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const ff = await root.create(genFeatureFlag())
                await unauth.delete(ff.name, opts)
            })
        }
    })
})
