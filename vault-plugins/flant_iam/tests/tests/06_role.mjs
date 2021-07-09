import { expect } from "chai"
import Faker from "faker"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genRoleCreatePayload, genRoleUpdatePayload, RoleAPI } from "./lib/role.mjs"

describe("Role", function () {
    const rootClient = getClient(rootToken)
    const root = new RoleAPI(rootClient)

    describe("payload", () => {
        describe("identifier", () => {
            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genRoleCreatePayload({ name: 100 }),
                    validateStatus: (x) => x == 201,
                },
                {
                    title: "absent name forbidden",
                    payload: (() => {
                        const p = genRoleUpdatePayload()
                        delete p.identifier
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
                    await root.create(x.payload, {
                        validateStatus: x.validateStatus,
                    })
                }),
            )
        })
    })

    async function createRole(override = {}) {
        const payload = genRoleCreatePayload({
            name: Faker.internet.domainWord(),
            ...override,
        })

        const { data: body } = await root.create(payload)

        return body.data
    }

    it("can be created", async () => {
        const data = await createRole()

        expect(data).to.exist.and.to.include.key("data")
        expect(data.role).to.include.keys(
            "name",
            "description",
            "scope",
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
        const createBodies = await Promise.all([createRole(), createRole(), createRole()])

        const names = createBodies.map((b) => b.data.name)

        const readBodies = await Promise.all(names.map((name) => root.read(name)))

        for (let i = 0; i < readBodies.length; i++) {
            const created = createBodies[i]
            const read = readBodies[i].data
            expect(read.data).to.deep.eq(created.data)
        }
        // expect(resp2.data).to.contain({ ...payload2, name: id2 })
        // expect(resp3.data).to.contain({ ...payload3, name: id3 })
    })

    it("can be updated", async () => {
        const payload = genRoleCreatePayload()
        const created = await createRole(payload)

        const name = created.data.name
        payload.description = Faker.lorem.sentence()
        delete payload.name

        // update
        const { data: updated } = await root.update(name, payload)

        // read
        const { data: read } = await root.read(name)
        const role = read.data

        expect(role).to.deep.eq({ ...payload, name, included_roles: null }, "payload must be saved")
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

        expect(data.data).to.be.an("object").and.to.have.key("names")
        expect(data.data.names).to.include(name)
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
                await unauth.create(genRoleCreatePayload(), opts)
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list(opts)
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const data = await createRole()
                await unauth.read(data.role.name, opts)
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const data = await createRole()
                await unauth.update(
                    data.role.name,
                    genRoleUpdatePayload({ type: data.role.type }),
                    opts,
                )
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const data = await createRole()
                await unauth.delete(data.role.name, opts)
            })
        }
    })
})
