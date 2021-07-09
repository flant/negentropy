import { expect } from "chai"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    genMultipassPayload,
    genPasswordPayload,
    genServiceAccountPayload,
    genTenantPayload,
} from "./lib/payloads.mjs"

//    /tenant/{tid}/service_account/{sa.uuid}

describe("Service Account", function () {
    const rootClient = getClient(rootToken)

    const rootTenantAPI = new API(
        rootClient,
        new EndpointBuilder(["tenant"]),
        new SingleFieldReponseMapper("data.tenant", "data.uuids"),
    )

    function getAPIClient(client) {
        return new API(
            client,
            new EndpointBuilder(["tenant", "service_account"]),
            new SingleFieldReponseMapper("data.service_account", "data.uuids"),
        )
    }

    const rootServiceAccountClient = getAPIClient(rootClient)

    function genPayload(override) {
        return genServiceAccountPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        return await rootTenantAPI.create({ payload })
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createServiceAccount(tid) {
        const payload = genPayload()
        return await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const sa = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload: genPayload(),
        })

        expect(sa).to.include.keys("uuid", "tenant_uuid", "resource_version")
        expect(sa.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(sa.tenant_uuid).to.eq(tid)
        expect(sa.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        // create
        const payload = genPayload()
        const created = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })

        const generated = {
            uuid: created.uuid,
            tenant_uuid: created.tenant_uuid,
            resource_version: created.resource_version,
            full_identifier: payload.identifier + "@serviceaccount." + tenant.identifier,
            origin: "iam",
            extensions: null,
        }

        // read
        const read = await rootServiceAccountClient.read({
            params: { tenant: tid, service_account: created.uuid },
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
        const created = await createServiceAccount(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, service_account: created.uuid }
        const updated = await rootServiceAccountClient.update({
            params,
            payload,
        })

        // read
        const read = await rootServiceAccountClient.read({ params })

        expect(read).to.deep.eq(updated)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const sa = await createServiceAccount(tid)

        // delete
        const params = { tenant: tid, service_account: sa.uuid }
        await rootServiceAccountClient.delete({ params })

        // read
        await rootServiceAccountClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const sa = await createServiceAccount(tid)

        // delete
        const params = { tenant: tid }
        const list = await rootServiceAccountClient.list({ params })

        expect(list).to.be.an("array").of.length(1) // if not 1, maybe serviceAccounts are not filtered by tenants
        expect(list[0]).to.eq(sa.uuid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const sa = await createServiceAccount(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, service_account: sa.uuid }
        const opts = expectStatus(404)
        await rootServiceAccountClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { service_account: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootServiceAccountClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootServiceAccountClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootServiceAccountClient.delete({ params, opts })
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

            beforeEach("create serviceAccount payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const sa = await rootServiceAccountClient.create({
                    params,
                    payload,
                })

                await unauth.read({
                    params: { ...params, service_account: sa.uuid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const sa = await rootServiceAccountClient.create({
                    params,
                    payload,
                })

                await unauth.update({
                    params: { ...params, service_account: sa.uuid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const sa = await rootServiceAccountClient.create({
                    params,
                    payload,
                })

                await unauth.delete({
                    params: { ...params, service_account: sa.uuid },
                    opts,
                })
            })
        }
    })

    describe("multipass", function () {
        const endpointBuilder = new EndpointBuilder(["tenant", "service_account", "multipass"])
        const rootMPClient = new API(
            rootClient,
            endpointBuilder,
            new SingleFieldReponseMapper("data.multipass", "data.uuids"),
        )

        async function createMultipass(t, u, override = {}) {
            const payload = genMultipassPayload(override)
            const params = { tenant: t, service_account: u }
            return await rootMPClient.create({ params, payload })
        }

        it("can be created", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)

            await createMultipass(t.uuid, u.uuid)
        })

        it("contains expected data when created", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)

            const mp = await createMultipass(t.uuid, u.uuid)

            expect(mp)
                .to.be.an("object")
                .and.include.keys(
                    "allowed_cidrs",
                    "allowed_roles",
                    "description",
                    "max_ttl",
                    "owner_type",
                    "owner_uuid",
                    "tenant_uuid",
                    "origin",
                    "extensions",
                    "ttl",
                    "uuid",
                    "valid_till",
                )
                .and.not.include.keys("salt")

            expect(mp.uuid, "uuid").to.be.a("string")
            expect(mp.owner_type, "owner_type").to.be.a("string")
            expect(mp.owner_uuid, "owner_uuid").to.be.a("string")
            expect(mp.tenant_uuid, "tenant_uuid").to.be.a("string")

            expect(mp.salt, "salt").to.be.undefined

            expect(mp.description, "description").to.be.a("string")

            expect(mp.allowed_cidrs, "allowed_cidrs").to.be.an("array")
            expect(mp.allowed_roles, "allowed_roles").to.be.an("array")

            expect(mp.ttl, "ttl").to.be.a("number")
            expect(mp.max_ttl, "max_ttl").to.be.a("number")
            expect(mp.valid_till, "valid_till")
                .to.be.a("number")
                .greaterThan(Date.now() / 1e3)
        })

        it("can be read", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                service_account: u.uuid,
                multipass: created.uuid,
            }
            const read = await rootMPClient.read({ params })

            expect(read).to.deep.eq(created)
        })

        it("can be listed", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)

            const createId = () => createMultipass(t.uuid, u.uuid).then((mp) => mp.uuid)

            const ids = await Promise.all([createId(), createId(), createId()])

            const params = {
                tenant: t.uuid,
                service_account: u.uuid,
            }

            const list = await rootMPClient.list({ params })

            expect(list).to.have.all.members(ids)
        })

        it("can be deleted", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                service_account: u.uuid,
                multipass: created.uuid,
            }

            await rootMPClient.delete({ params })

            await rootMPClient.read({ params, opts: expectStatus(404) })
        })

        it("cannot be updated", async () => {
            const t = await createTenant()
            const u = await createServiceAccount(t.uuid)
            const createdMP = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                service_account: u.uuid,
                multipass: createdMP.uuid,
            }

            await rootMPClient.update({
                params,
                payload: genMultipassPayload(),
                opts: expectStatus(405),
            })
        })
    })

    describe("password", function () {
        const endpointBuilder = new EndpointBuilder(["tenant", "service_account", "password"])
        const rootPasswordClient = new API(
            rootClient,
            endpointBuilder,
            new SingleFieldReponseMapper("data.password", "data.uuids"),
        )

        async function createPassword(t, sa, override = {}) {
            const payload = genPasswordPayload(override)
            const params = { tenant: t, service_account: sa }
            return await rootPasswordClient.create({
                params,
                payload,
            })
        }

        it("can be created", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            await createPassword(t.uuid, sa.uuid)
        })

        it("contains expected data when created", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            const password = await createPassword(t.uuid, sa.uuid)

            expect(password)
                .to.be.an("object")
                .and.include.keys(
                    "uuid",
                    "owner_uuid",
                    "tenant_uuid",
                    "description",
                    "allowed_cidrs",
                    "allowed_roles",
                    "ttl",
                    "valid_till",
                    "secret",
                )

            expect(password.uuid, "uuid").to.be.a("string")
            expect(password.owner_uuid, "owner_uuid").to.be.a("string")
            expect(password.tenant_uuid, "tenant_uuid").to.be.a("string")

            expect(password.secret, "secret length")
                .to.be.a("string")
                .with.length.greaterThanOrEqual(20)

            expect(password.description, "description").to.be.a("string")

            expect(password.allowed_cidrs, "allowed_cidrs").to.be.an("array")
            expect(password.allowed_roles, "allowed_roles").to.be.an("array")

            expect(password.ttl, "ttl").to.be.a("number")
            expect(password.valid_till, "valid_till")
                .to.be.a("number")
                .greaterThan(Date.now() / 1e3)
        })

        it("can be read", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)
            const password = await createPassword(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
                password: password.uuid,
            }
            const read = await rootPasswordClient.read({ params })

            // sensitive data is returned only on creation, so we delete it before deep comparison
            delete password.secret
            expect(read).to.deep.eq(password)
        })

        it("can be listed", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            const createId = () => createPassword(t.uuid, sa.uuid).then((password) => password.uuid)

            const ids = await Promise.all([createId(), createId(), createId()])

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
            }

            const list = await rootPasswordClient.list({ params })

            expect(list).to.have.all.members(ids)
        })

        it("can be deleted", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)
            const password = await createPassword(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
                password: password.uuid,
            }

            await rootPasswordClient.delete({ params })

            await rootPasswordClient.read({ params, opts: expectStatus(404) })
        })

        it("cannot be updated", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)
            const password = await createPassword(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
                password: password.uuid,
            }

            await rootPasswordClient.update({
                params,
                payload: genMultipassPayload(),
                opts: expectStatus(405),
            })
        })
    })
})
