// @ts-expect-error
import { deleteDownloadsFolder } from "cypress-delete-downloads-folder";

describe("Builds", () => {
  beforeEach(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
    cy.login("tester");
    cy.visit("/");
    cy.get("#show-add-repo").click();
    cy.get("#url").type(`${Cypress.env("host")}/repository.git`, {
      force: true,
    });
    cy.get("#add-repo").click({ force: true });
    cy.get("#show-add-build").click();
  });
  afterEach(() => {
    cy.visit("/");
    cy.get(".fa-trash", { timeout: 5000 }).click();
    cy.contains(".git").should("not.exist");
  });
  it("Tests single build", () => {
    cy.fixture("single.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
    cy.get('[href="/build/1/container/single"]').click({ force: true });
    cy.contains("ci.json").should("be.visible");
  });
  it("Tests multiple builds", () => {
    cy.fixture("dependency.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get('[href="/build/1/container/first"]')
      .parent()
      .get(".bg-yellow-500", { timeout: 20000 })
      .should("be.visible");
    cy.get('[href="/build/1/container/second"]')
      .parent()
      .get(".bg-gray-500")
      .should("be.visible");
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
  });
  it("Tests needing files", () => {
    cy.fixture("neededFile.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-green-500", { timeout: 20000 }).should("have.length", 2);
    cy.contains("first").parent().contains("/repo").click({ force: true });
    cy.readFile(`${Cypress.config("downloadsFolder")}/_repo.tar`);
    deleteDownloadsFolder();
    cy.get('[href="/build/1/container/second"]').click({ force: true });
    cy.contains("ci.json").should("be.visible");
  });
  it("Tests environment variables", () => {
    cy.fixture("environment.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
    cy.get('[href="/build/1/container/environment"]').click({ force: true });
    cy.contains("Hello World").should("be.visible");
  });
  it("Tests service containers", () => {
    cy.fixture("service.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
    cy.get('[href="/build/1/container/service"]').click({ force: true });
    cy.contains("HelloWorld").should("be.visible");
  });
  it("Tests log websocket", () => {
    cy.fixture("delayedLog.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get("[href='/build/1/container/single']").click({ force: true });
    cy.contains("first", { timeout: 20000 }).should("be.visible");
    cy.contains("second").should("not.exist");
    cy.contains("second", { timeout: 20000 }).should("be.visible");
  });
  it("Tests persistent containers", () => {
    cy.fixture("persist.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.wait(5000);
    cy.contains<HTMLElement>("Preview Link").then(($a) => {
      const href = $a.prop("href");
      cy.request(href).its("body").should("include", "OK");
    });
    cy.get(".fa-stop").click();
    cy.get(".bg-red-500", { timeout: 20000 }).should("be.visible");
  });
});
