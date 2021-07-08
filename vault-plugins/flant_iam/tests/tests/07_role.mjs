import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genRoleUpdatePayload, RoleAPI } from "./lib/role.mjs"
import { expect } from "chai"
import Faker from "faker"

describe("Role", function () {
    const rootClient = getClient(rootToken)
    const root = new RoleAPI(rootClient)

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`role/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })

    describe("payload", () => {
        describe("identifier", () => {
            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genRoleUpdatePayload({ name: 100 }),
                    validateStatus: (x) => x == 201,
                },
                {
                    title: "absent identifier forbidden",
                    payload: (() => {
                        const p = genRoleUpdatePayload({})
                        delete p.identifier
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genRoleUpdatePayload({ identifier: "" }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "array forbidden",
                    payload: genRoleUpdatePayload({ identifier: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genRoleUpdatePayload({ identifier: { a: 1 } }),
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

    async function createRole(override = {}) {
        const payload = genRoleUpdatePayload({
            name: Faker.internet.domainWord(),
            ...override,
        })

        const { data: body } = await root.create(payload)

        return body
    }

    it("can be created", async () => {
        const body = await createRole()

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.include.keys(
            "name",
            "description",
            "type",
            "options_schema",
            "require_one_of_feature_flags",
        )
    })

    it("can be read", async () => {
        const created = await createRole()

        const { data: read } = await root.read(created.data.name)

        expect(read.data).to.deep.eq(created.data)
    })

    it("can be read by name", async () => {
        const createBodies = await Promise.all([
            createRole(),
            createRole(),
            createRole(),
        ])

        const names = createBodies.map((b) => b.data.name)

        const readBodies = await Promise.all(
            names.map((name) => root.read(name)),
        )

        for (let i = 0; i < readBodies.length; i++) {
            const created = createBodies[i]
            const read = readBodies[i].data
            expect(read.data).to.deep.eq(created.data)
        }
        // expect(resp2.data).to.contain({ ...payload2, name: id2 })
        // expect(resp3.data).to.contain({ ...payload3, name: id3 })
    })

    it("can be updated", async () => {
        const payload = genRoleUpdatePayload()
        const created = await createRole(payload)

        const name = created.data.name
        payload.description = Faker.lorem.sentence()

        // update
        const { data: updated } = await root.update(name, payload)

        // read
        const { data: read } = await root.read(name)
        const role = read.data

        expect(role).to.deep.eq(
            { ...payload, name, included_roles: null },
            "payload must be saved",
        )
    })

    it("can be deleted", async () => {
        const created = await createRole()
        const name = created.data.name

        await root.delete(name)

        await root.read(name, expectStatus(404))
    })

    it("can be listed", async () => {
        const created = await createRole()
        const name = created.data.name

        const { data } = await root.list()

        expect(data.data).to.be.an("object")
    })

    it("has identifying fields in list", async () => {
        const created = await createRole()
        const name = created.data.name

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("names")
        expect(listBody.data.names).to.include(name)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)

        it("cannot read, gets 404", async () => {
            await root.read("no-such", opts)
        })

        it("cannot update, gets 404", async () => {
            await root.update("no-such", genRoleUpdatePayload(), opts)
        })

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
            const unauth = new RoleAPI(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create(genRoleUpdatePayload(), opts)
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list(opts)
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const body = await createRole()
                await unauth.read(body.data.name, opts)
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const body = await createRole()
                await unauth.update(
                    body.data.name,
                    genRoleUpdatePayload({ type: body.data.type }),
                    opts,
                )
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const body = await createRole()
                await unauth.delete(body.data.name, opts)
            })
        }
    })
})
