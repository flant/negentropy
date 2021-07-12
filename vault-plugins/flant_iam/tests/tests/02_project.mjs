import { expect } from "chai"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genProjectPayload, genTenantPayload } from "./lib/payloads.mjs"

//    /tenant/{tid}/project/{pid}

describe("Project", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(
        rootClient,
        new EndpointBuilder(["tenant"]),
        new SingleFieldReponseMapper("data.tenant", "data.uuids"),
    )
    function getAPIClient(client) {
        return new API(
            client,
            new EndpointBuilder(["tenant", "project"]),
            new SingleFieldReponseMapper("data.project", "data.uuids"),
        )
    }

    const rootProjectAPI = getAPIClient(rootClient)

    function genPayload(override) {
        return genProjectPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        return await rootTenantAPI.create({ payload })
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createProject(tid) {
        const payload = genPayload()
        return await rootProjectAPI.create({
            params: { tenant: tid },
            payload,
        })
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const project = await rootProjectAPI.create({
            params: { tenant: tid },
            payload: genPayload(),
        })

        expect(project).to.include.keys("uuid", "tenant_uuid", "resource_version")
        expect(project.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(project.tenant_uuid).to.eq(tid)
        expect(project.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        // create
        const payload = genPayload()
        const created = await rootProjectAPI.create({
            params: { tenant: tid },
            payload,
        })
        const pid = created.uuid
        const generated = {
            uuid: created.uuid,
            tenant_uuid: created.tenant_uuid,
            resource_version: created.resource_version,
        }

        // read
        const read = await rootProjectAPI.read({
            params: { tenant: tid, project: pid },
        })

        expect(read).to.deep.eq({ ...payload, ...generated }, "must have generated fields")
        expect(read).to.deep.eq(
            created,
            "reading and creation responses should contain the same data",
        )
        expect(read.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createProject(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, project: created.uuid }
        const updated = await rootProjectAPI.update({
            params,
            payload,
        })

        // read
        const read = await rootProjectAPI.read({ params })

        expect(read).to.deep.eq(updated)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const project = await createProject(tid)
        const pid = project.uuid

        // delete
        const params = { tenant: tid, project: pid }
        await rootProjectAPI.delete({ params })

        // read
        await rootProjectAPI.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const project = await createProject(tid)
        const pid = project.uuid

        // delete
        const params = { tenant: tid }
        const list = await rootProjectAPI.list({ params })

        expect(list).to.be.an("array").of.length(1) // if not 1, maybe projects are not filtered by tenants
        expect(list[0]).to.eq(pid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const project = await createProject(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, project: project.uuid }
        const opts = expectStatus(404)
        await rootProjectAPI.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { project: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootProjectAPI.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootProjectAPI.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootProjectAPI.delete({ params, opts })
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
            const unauth = getAPIClient(client)
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create project payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const project = await rootProjectAPI.create({
                    params,
                    payload,
                })
                const pid = project.uuid

                await unauth.read({
                    params: { ...params, project: pid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const project = await rootProjectAPI.create({
                    params,
                    payload,
                })
                const pid = project.uuid
                await unauth.update({
                    params: { ...params, project: pid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const project = await rootProjectAPI.create({
                    params,
                    payload,
                })
                const pid = project.uuid
                await unauth.delete({
                    params: { ...params, project: pid },
                    opts,
                })
            })
        }
    })
})
