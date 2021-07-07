import { expect } from "chai"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { FeatureFlagAPI, genFeatureFlag } from "./lib/featureflag.mjs"

describe("Feature flag", function () {
    const rootClient = getClient(rootToken)
    const root = new FeatureFlagAPI(rootClient)

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`featureFlag/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })

    describe("payload", () => {
        describe("name", () => {
            after("clean", async () => {
                const { data } = await root.list()
                const deletions = data.data.names.map((name) =>
                    root.delete(name),
                )
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

        const { data: body } = await root.create(payload, expectStatus(201))

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.include.keys("name")
        expect(body.data.name).to.eq(payload.name)
    })

    it("can be listed", async () => {
        const payload = genFeatureFlag()
        await root.create(payload)

        const { data } = await root.list()

        expect(data.data).to.be.an("object")
    })

    it("has identifying fields in list", async () => {
        const payload = genFeatureFlag()
        const { data: creationBody } = await root.create(payload)
        const name = creationBody.data.name

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("names")
        expect(listBody.data.names).to.include(name)
    })

    it("can be deleted", async () => {
        const createPld = genFeatureFlag()

        const { data: created } = await root.create(createPld)
        const name = created.data.name

        await root.delete(created.data.name)

        const { data: listBody } = await root.list()
        expect(listBody.data.names).to.not.include(name)
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
                const { data } = await root.create(genFeatureFlag())
                await unauth.delete(data.data.name, opts)
            })
        }
    })
})
