import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import { genUserPayload, SubTenantEntrypointBuilder } from "./lib/subtenant.mjs"
import { API } from "./lib/api.mjs"

/*
 * Every typical CRUDL under tenant
 *          /tenant/{tid}/_subtenant_/{subid}
 */
const subtenants = [
    {
        name: "user",
        genPayload: genUserPayload,
    },
    // {
    //     name: "project",
    //     genPayload: genProjectPayload,
    // },
    // {
    //     name: "group",
    //     genPayload: genGroupPayload,
    // },
    // {
    //     name: "service_account",
    //     genPayload: genServiceAccountPayload,
    // },
    // {
    //     name: "role",
    //     genPayload: genRolePayload,
    // },
]

describe("Subtenant", function () {
    // TODO: add tests for fields that are unique within a tenant
    for (const subtenant of subtenants) {
        describe(subtenant.name, function () {
            const rootClient = getClient(rootToken)
            const rootTenantAPI = new API(
                rootClient,
                new TenantEndpointBuilder(),
            )

            const entrypointBuilder = new SubTenantEntrypointBuilder(
                subtenant.name,
            )
            const rootSubtenantAPI = new API(rootClient, entrypointBuilder)

            function genPayload() {
                return subtenant.genPayload()
            }

            async function createTenantId() {
                const payload = genTenantPayload()
                const { data } = await rootTenantAPI.create({ payload })
                return data.data.uuid
            }

            async function createSubtenantId(tid) {
                const payload = genPayload()
                const { data: body } = await rootSubtenantAPI.create({
                    params: { tenant: tid },
                    payload,
                })
                return body.data.uuid
            }

            it("can be created", async () => {
                const tid = await createTenantId()

                const { data: body } = await rootSubtenantAPI.create({
                    params: { tenant: tid },
                    payload: genPayload(),
                })

                expect(body).to.exist.and.to.include.key("data")
                expect(body.data).to.have.key("uuid")
                expect(body.data.uuid).to.be.a("string").of.length.above(10)
            })

            it("can be read", async () => {
                const tid = await createTenantId()

                // create
                const payload = genPayload()
                const { data: body } = await rootSubtenantAPI.create({
                    params: { tenant: tid },
                    payload,
                })
                const id = body.data.uuid

                // read
                const { data: user } = await rootSubtenantAPI.read({
                    params: { tenant: tid, [subtenant.name]: id },
                })
                expect(user.data).to.deep.eq({ uuid: id, tenant_uuid: tid })
            })

            it("can be updated", async () => {
                const tid = await createTenantId()

                // create
                const subid = await createSubtenantId(tid)

                // update
                const payload = genPayload()
                const params = { tenant: tid, user: subid }
                await rootSubtenantAPI.update({ params, payload })

                // read
                const { data: body } = await rootSubtenantAPI.read({ params })
                const sub = body.data

                expect(sub).to.deep.eq({ uuid: subid, tenant_uuid: tid })
            })

            it("can be deleted", async () => {
                const tid = await createTenantId()

                // create
                const subid = await createSubtenantId(tid)

                // delete
                const params = { tenant: tid, [subtenant.name]: subid }
                await rootSubtenantAPI.delete({ params })

                // read
                await rootSubtenantAPI.read({ params, opts: expectStatus(404) })
            })

            it("can be listed", async () => {
                // create
                const tid = await createTenantId()
                const subid = await createSubtenantId(tid)

                // delete
                const params = { tenant: tid }
                const { data: body } = await rootSubtenantAPI.list({ params })

                expect(body.data).to.be.an("object").and.include.keys("uuids")
                expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe users are not filtered by tenants
                expect(body.data.uuids[0]).to.eq(subid)
            })

            describe("when does not exist", () => {
                const opts = expectStatus(404)
                const params = { [subtenant.name]: "no-such" }

                before("create tenant", async () => {
                    params.tenant = await createTenantId()
                })

                it("cannot read, gets 404", async () => {
                    await rootSubtenantAPI.read({ params, opts })
                })

                it("cannot update, gets 404", async () => {
                    await rootSubtenantAPI.update({
                        params,
                        opts,
                        payload: genTenantPayload(),
                    })
                })

                it("cannot delete, gets 404", async () => {
                    await rootSubtenantAPI.delete({ params, opts })
                })
            })

            describe("access", function () {
                describe("when unauthenticated", function () {
                    runWithClient(getClient(), 400)
                })

                describe("when unauthorized", function () {
                    runWithClient(getClient("xxx"), 403)
                })

                function runWithClient(client, expectedStatus) {
                    const opts = expectStatus(expectedStatus)
                    const unauth = new API(client, entrypointBuilder)
                    let payload = {}

                    const params = {}
                    before("create tenant", async () => {
                        params.tenant = await createTenantId()
                    })

                    beforeEach("create user payload", async () => {
                        payload = genPayload()
                    })

                    it(`cannot create, gets ${expectedStatus}`, async () => {
                        await unauth.create({ params, payload, opts })
                    })

                    it(`cannot list, gets ${expectedStatus}`, async () => {
                        await unauth.list({ params, opts })
                    })

                    it(`cannot read, gets ${expectedStatus}`, async () => {
                        const { data } = await rootSubtenantAPI.create({
                            params,
                            payload,
                        })
                        const subid = data.data.uuid

                        await unauth.read({
                            params: { ...params, [subtenant.name]: subid },
                            opts,
                        })
                    })

                    it(`cannot update, gets ${expectedStatus}`, async () => {
                        const { data } = await rootSubtenantAPI.create({
                            params,
                            payload,
                        })
                        const subid = data.data.uuid
                        await unauth.update({
                            params: { ...params, [subtenant.name]: subid },
                            payload,
                            opts,
                        })
                    })

                    it(`cannot delete, gets ${expectedStatus}`, async () => {
                        const { data } = await rootSubtenantAPI.create({
                            params,
                            payload,
                        })
                        const subid = data.data.uuid
                        await unauth.delete({
                            params: { ...params, [subtenant.name]: subid },
                            opts,
                        })
                    })
                }
            })
        })
    }
})
