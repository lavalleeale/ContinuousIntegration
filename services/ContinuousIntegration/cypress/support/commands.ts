// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add('login', (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add('drag', { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add('dismiss', { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite('visit', (originalFn, url, options) => { ... })
Cypress.Commands.add("login", (name: string) => {
  cy.session(`Login-${name}`, () => {
    cy.clearAllCookies();
    cy.visit("http://localhost:8080");
    cy.url().should("contain", "/login");
    cy.get("#username").type(name);
    cy.get("#password").type(name);
    cy.get("#login").click();
    cy.url().should("eq", "http://localhost:8080/");
  });
});

Cypress.Commands.add("logout", () => {
  cy.session("Logout", () => {
    cy.visit("/login");
    cy.get("#logout").click();
    cy.url().should("eq", "http://localhost:8080/login");
  });
});
