describe("Basic Spec", () => {
  it("Runs Build", () => {
    cy.task("db:seed", "default");
    cy.visit("http://localhost:8080");
    cy.url().should("contain", "/login");
    cy.get("#username").type("tester");
    cy.get("#password").type("tester");
    cy.get("#login").click();
    cy.url().should("eq", "http://localhost:8080/");
    cy.get("#show-add-repo").click();
    cy.get("#url").type("https://github.com/sshca/sshca", { force: true });
    cy.get("#add-repo").click({ force: true });
    cy.get("#show-add-build").click();
    cy.fixture("sshca.json").then((sshca) => {
      cy.get("#command").type(JSON.stringify(sshca), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
      cy.get("#add-build").click({ force: true });
      cy.get(".bg-yellow-500").should("be.visible");
      cy.get(".bg-gray-500").should("be.visible");
      cy.get(".bg-green-500", { timeout: 100000 }).should("be.visible");
      cy.get(".bg-yellow-500", { timeout: 100000 }).should("be.visible");
      cy.get(".bg-green-500", { timeout: 1000000 }).should("have.length", 2);
    });
  });
});
