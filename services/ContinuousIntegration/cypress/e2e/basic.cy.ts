// @ts-expect-error
import { deleteDownloadsFolder } from 'cypress-delete-downloads-folder';

describe("Full Spec", {testIsolation:false}, () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Logs In", () => {
    cy.visit("/");
    cy.login("tester");
  });
  it("Creates Repo", () => {
    cy.get("#show-add-repo").click();
    cy.get("#url").type("https://github.com/lavalleeale-tests/test", { force: true });
    cy.get("#add-repo").click({ force: true });
  });
  it("Creates Build", () => {
    cy.get("#show-add-build").click();
    cy.fixture("basic.json").then((test) => {
      cy.get("#command").type(JSON.stringify(test), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
      cy.get("#add-build").click({ force: true });
    });
  });
  it("Runs Build", () => {
    cy.get(".bg-yellow-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-gray-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("have.length", 2);
    cy.contains("/repo").click()
    cy.readFile(`${Cypress.config('downloadsFolder')}/_repo.tar`)
    deleteDownloadsFolder()
    cy.contains("go").click();
    cy.contains("/neededFiles/repo/test").should("be.visible");
  });

  it("Deletes Repo", () => {
    cy.login("tester");
    cy.visit("/");
    cy.get(".fa-trash").click();
    cy.contains("test").should("not.exist");
  });
});
