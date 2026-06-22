# Salescraft Autonomous SDLC Evaluation Rubric

This rubric maps final trial evaluation to the research plan for autonomous SDLC
model and toolchain selection.

## Research Questions

- RQ0: Which model/toolchain allocation maximizes build quality while minimizing
  cost?
- RQ1: How does the planning model affect downstream build quality?
- RQ2: How does the code generation model affect correctness and
  cost-to-completion?
- RQ3: Does a different test model improve defect detection?
- RQ4: How does the orchestrator affect build success rate and cost?
- RQ5: Does a multi-model pipeline outperform a single model?
- RQ6: How does context passing affect pipeline quality?
- RQ7: At what application complexity threshold do cheaper models fail?
- RQ8: What is the cost-quality Pareto frontier?
- RQ9: How does loop strategy affect convergence?
- RQ10: Can models maintain domain-driven design discipline?

## Quality Score

The evaluator records raw data and derives a weighted 0-100 quality score:

- Functional correctness: 35
- DDD adherence: 20
- Code quality: 15
- Completeness: 15
- Security: 5
- Documentation: 5
- Performance: 5

## Functional Correctness

Use fixed verification commands and browser workflow tests. Passing unit,
integration, typecheck, lint, and build gates are necessary but not sufficient:
the generated app should also launch and expose usable product workflows.

## DDD Adherence

Score use of ubiquitous language, bounded contexts, aggregates/value objects,
and domain events. Penalize generic CRUD naming that replaces commercial
flooring concepts.

Required domain language includes:

- Specification
- Takeoff
- Wear Layer
- Seaming
- Transition Strip
- Punch List
- General Contractor
- Architect of Record
- Material Safety Data Sheet / MSDS
- Lead Time
- Floor Prep
- Attic Stock

Expected bounded contexts:

- Sales
- Product Catalog
- Project Management
- Marketing
- Recommendations / AI

## Report Requirements

The final report should explicitly answer:

- Whether the trial succeeded end to end.
- How much of the app was completed.
- What verification and browser workflows passed or failed.
- Whether the generated architecture preserves domain language and bounded
  contexts.
- What the result implies for the active research question and independent
  variable.
- What residual risks remain before calling the app production-quality.
