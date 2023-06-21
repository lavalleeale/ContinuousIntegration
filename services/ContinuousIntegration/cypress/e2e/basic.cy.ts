// @ts-expect-error
import { deleteDownloadsFolder } from "cypress-delete-downloads-folder";

describe("Full Spec", { testIsolation: false }, () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Tests basic build", () => {
    cy.visit("/");
    cy.login("tester");
    cy.get("#show-add-repo").click();
    cy.get("#url").type(`${Cypress.env("host")}/repository.git`, {
      force: true,
    });
    cy.get("#add-repo").click({ force: true });
    cy.get("#show-add-build").click();
    cy.fixture("basic.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
    });
    cy.get("#add-build").click({ force: true });
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.get(".bg-gray-500", { timeout: 20000 }).should("have.length", 2);
    cy.get(".bg-green-500", { timeout: 20000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.contains("go").click();
    cy.contains("/neededFiles/repo/ci.json").should("not.exist");
    cy.contains("CYPRESS").should("not.exist");
    cy.contains("/neededFiles/repo/ci.json", { timeout: 20000 }).should(
      "be.visible"
    );
    cy.contains("CYPRESS").should("be.visible");
    cy.go("back");
    cy.get(".bg-green-500", { timeout: 20000 }).should("have.length", 2);
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.contains("assets").parent().contains("/repo").click();
    cy.readFile(`${Cypress.config("downloadsFolder")}/_repo.tar`);
    deleteDownloadsFolder();
    cy.get(".bg-green-500", { timeout: 20000 }).should("have.length", 2);
    cy.get(".bg-yellow-500", { timeout: 20000 }).should("be.visible");
    cy.contains<HTMLElement>("Preview Link").then(($a) => {
      const href = $a.prop("href");
      cy.request(href).its("body").should("include", "OK");
    });
    cy.get(".fa-stop").click();
    cy.wait(1000);
    cy.reload();
    cy.get(".bg-red-500", { timeout: 20000 }).should("be.visible");
    cy.login("tester");
    cy.visit("/");
    cy.get(".fa-trash").click();
    cy.contains("test").should("not.exist");
  });
});
