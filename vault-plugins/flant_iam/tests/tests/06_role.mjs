import { expect } from "chai"
import Faker from "faker"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genRoleCreatePayload } from "./lib/payloads.mjs"

describe("Role", function() {
    const rootClient = getClient(rootToken)

    function getAPIClient(client) {
        return new API(
            client,
            new EndpointBuilder(["role"]),
            new SingleFieldReponseMapper("data.role", "data.names"),
        )
    }

    const root = getAPIClient(rootClient)

    describe("payload", () => {
        describe("name", () => {
            after("clean", async () => {
                const names = await root.list()
                const deletions = names.map((role) => root.delete({ params: { role: role.name } }))
                await Promise.all(deletions)
            })

            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genRoleCreatePayload({
                        name: Math.round(Math.random() * 1e9),
                    }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent name forbidden",
                    payload: (() => {
                        const p = genRoleCreatePayload({})
                        delete p.name
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genRoleCreatePayload({ name: "" }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "array forbidden",
                    payload: genRoleCreatePayload({ name: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genRoleCreatePayload({ name: { a: 1 } }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
            ]

            invalidCases.forEach((x) =>
                it(x.title, async () => {
                    await root.create({
                        payload: x.payload,
                        opts: {
                            validateStatus: x.validateStatus,
                        },
                    })
                }),
            )
        })
    })

    it("can be created", async () => {
        const payload = genRoleCreatePayload()

        const role = await root.create({ payload })

        expect(role).to.include.keys(
            "name",
            "description",
            "scope",
            "options_schema",
            "require_one_of_feature_flags",
        )
        expect(role.name).to.eq(payload.name)
    })

    it("can be listed", async () => {
        const payload = genRoleCreatePayload()
        await root.create({ payload })

        const names = await root.list()

        expect(names).to.be.an("array")
    })

    it("has identifying fields in list", async () => {
        const payload = genRoleCreatePayload()
        const role = await root.create({ payload })

        const list = await root.list()

        expect(list.map(r => r.name)).to.include(role.name)
    })

    it("can be deleted", async () => {
        const payload = genRoleCreatePayload()

        const role = await root.create({ payload })

        const params = { role: role.name }
        await root.delete({ params })

        const list = await root.list()
        expect(list).to.not.include(role.name)
    })

    async function createRole(override = {}) {
        const payload = genRoleCreatePayload({
            name: Faker.internet.domainWord(),
            ...override,
        })

        return await root.create({ payload })
    }

    it("can be read by name", async () => {
        const createdList = await Promise.all([createRole(), createRole(), createRole()])

        const readList = await Promise.all(
            createdList.map((r) => root.read({ params: { role: r.name } })),
        )

        for (let i = 0; i < readList.length; i++) {
            const created = createdList[i]
            const read = readList[i]
            expect(read).to.deep.eq(created)
        }
    })

    it("can be updated", async () => {
        const payload = genRoleCreatePayload()
        const created = await createRole(payload)

        const name = created.name
        const params = { role: name }
        payload.description = Faker.lorem.sentence()
        delete payload.name

        // update
        const updated = await root.update({ params, payload })

        // read
        const read = await root.read({ params })

        expect(read).to.deep.eq({ ...payload, name, included_roles: null }, "payload must be saved")
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)

        it("cannot delete, gets 404", async () => {
            await root.delete({ params: { role: "no-such" }, opts })
        })
    })

    describe("no access", function() {
        describe("when unauthenticated", function() {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function() {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const unauth = getAPIClient(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                const payload = genRoleCreatePayload()
                await unauth.create({ payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ opts })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const payload = genRoleCreatePayload()
                const role = await root.create({ payload })
                const params = { role: role.name }
                await unauth.delete({ params, opts })
            })
        }
    })
})
