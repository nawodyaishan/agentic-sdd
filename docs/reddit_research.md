Agentic SDD (Spec-Driven Development) is an emerging approach where AI agents automate and manage the software development lifecycle, focusing on maintaining the original intent from concept to code. Redditors generally agree that while agents can handle code generation, the primary challenges and benefits lie in orchestrating multiple agents, managing governance, and ensuring consistent results.

Core Concepts and Benefits
Shifting Focus from Code to Orchestration: The conversation around Agentic SDD has moved beyond merely whether agents can write code to how to run a software organization where agents perform a significant portion of the work. This involves coordinating multiple agents and integrating them into existing engineering systems. "The conversation is no longer 'Can agents write code?' instead the conversation is becoming 'How do we run a software organization where agents are responsible for a meaningful percentage of the work?'"r/AI_Agents
.

Maintaining Intent Across the SDLC: Agentic SDD aims to ensure that the initial intent of a project is preserved throughout the development pipeline, from initial ideas to documentation, tickets, and code. This helps prevent miscommunication and ensures that the final product aligns with the original vision. "An Agentic SDLC : AI agents that sit across your entire dev pipeline and make sure nothing gets lost."r/AI_Agents
.

Enhanced Productivity and Focus: Many Redditors report increased productivity and satisfaction by leveraging agents to handle mundane coding tasks, allowing them to focus on higher-level problem-solving, architecture, and business objectives. One Redditor with ADHD found that agents significantly help with procrastination, providing a starting point for tasks within minutes. "to focus on one task at a time. So I took the task from Jira, solved it, pushed it, and took the next. Clear focus on one thing only. Right now I am not a developer by trade anymore, but quite often I still need to develop something, mostly prototyping for our software or such things. So I give the task to Claude; it starts coding, and while it does, I switch my focus to"r/ADHD_Programmers
.

Challenges and Considerations
Governance and Auditability: A significant challenge is establishing clear governance, audit trails, access control, and rollback mechanisms for agent-driven development. These organizational problems, often disguised as technical ones, become critical when issues arise, and teams struggle to identify who approved what or why an agent had certain access. "Governance, who approved this, why did this agent have access to that, what's the rollback path, is an organizational problem dressed up as a technical one, and most teams don't realize that until something goes wrong and they can't answer those questions."r/AI_Agents
.

ROI Justification and Cost Control: Organizations are struggling to justify the Return on Investment (ROI) for agentic development, especially as token spending can get out of control with open-ended agent sessions. Well-structured workflows are needed to complete tasks efficiently and cost-effectively. "IMO, the biggest question yet to be answered well is how to justify the ROI. Execs want to justify their spend in their budget and organizations are struggling to show it."r/AI_Agents
.

Evolving Standards and Tooling: The underlying capabilities of AI agents are still rapidly evolving, meaning that current standards and tools for agent orchestration might quickly become outdated. This rapid change makes it difficult to commit to a single framework. "Whatever becomes the standard now might look very different from what's needed in 18 months."r/AI_Agents
.

Practical Implementations
Multi-Agent Workflows: Some developers create sophisticated multi-agent systems where different AI models are used for specific phases of development, optimizing for capabilities like initial understanding, codebase exploration, proposing approaches, and detailed specification. This can reduce costs by using less powerful models for simpler tasks. "I use a 9-agent SDD harness where each phase uses a different model. The total cost is $10-15/month."r/cursor
.

Human-in-the-Loop Processes: While agents automate much of the work, human oversight remains crucial for reviewing plans, refining specifications, and ensuring quality. This involves treating AI as a junior developer whose work needs careful review and iteration. "Treat AI as a junior developer. You can trust AI to do the work quickly, but the quality, all the info, all the plans, technical decision is your job."r/ClaudeCode
.

Are you interested in exploring specific tools or frameworks for Agentic SDD?

AI Development Communities

SpecDrivenDevelopment
2.9K weekly visitors
Join
Spec-driven development is the evolution beyond vibe coding. Instead creating code from single prompt, you begin with a clear specification. This specification serves as a contract and single source of truth, guiding tools and AI agents to generate, test, and validate code.

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

cursor
101K weekly visitors
Join
The best way to code with AI - cursor.com

ClaudeCode
716K weekly visitors
Joined
a community for building, learning, and sharing with claude code.
ADHD_Programmers
25K weekly visitors
Join
Users are important! Without users, reddit would be little more than chunks of code on a server. We love reddit! We hope to make it a better place for everyone!
Generated from these posts:
1
 r/AI_Agents
12 months ago nobody understood why we were building Agentic SDLC. Now it feels like everyone is heading in the same direction.
17
·
18
2
 r/SpecDrivenDevelopment
Will the AI-assisted development model of SDD still be useful in the future? Is it already outdated?
5
·
6

+7
View all
Is this helpful?

Helpful

Unhelpful
Agentic SDD vs loop vs graph
Agentic SDD vs loop vs graph
r/LangChain
From r/AskProgramming, r/PromptEngineering, r/LangChain +7 more
Agentic SDD (Spec-Driven Development) often leverages different agent architectures, with the primary distinction being between agent loops and graph-based architectures. While loops involve a single agent iteratively performing tasks, graph-based approaches orchestrate multiple specialized agents in a defined workflow, which Redditors generally find more robust for complex problems.

Agent Loops
Sequential Execution: Agent loops involve a single agent that continuously reasons, picks a tool, reads results, and decides the next action until a task is complete. The entire orchestration logic, persona for sub-tasks, formatting rules, and exit criteria are typically contained within a single system prompt. "One prompt instructs the model to reason about the task, pick a tool call, read the result, then decide what to do next, over and over until it judges the job done."r/PromptEngineering

Context Management: In a standard agent loop, every tool result is appended to the same context window, meaning the input grows with each step. This can lead to non-deterministic tool selection and makes debugging difficult due to the lack of isolated steps. "Each tool result gets appended into the same context window, so the input grows with every step."r/PromptEngineering

Simplicity for Basic Tasks: For smaller, less complex workflows, a simple loop with one agent is often sufficient and easier to implement. "For small workflows, a simple loop with one agent is usually enough."r/vibecoding

Graph-Based Architectures
Orchestrated Specialization: Graph-based architectures move control flow into code, representing branching as a graph of nodes and edges, similar to a state machine. Each node can be a specialized agent with a specific job and its own small, focused prompt. "The branching becomes a graph of nodes and edges, closer to a state machine than a block of prose. Each node gets its own small prompt with one job."r/PromptEngineering

Improved Debugging and Reproducibility: With graph engineering, state becomes an explicit object rather than being inferred from a growing context. This allows for isolated inspection of steps and makes the execution path more reproducible, as routing can be based on deterministic comparisons. "Debugging agents built this way is hard because there is no isolated step to inspect... The same input takes the same path every time, and the reproducibility problem you describe stops existing."r/PromptEngineering

Efficiency and Cost-Effectiveness: Graph workflows can be more efficient in terms of token usage and wall-clock time compared to traditional agent loops, especially for tasks involving large inputs. This is because specialized nodes can use smaller, less expensive models for specific tasks. "Accuracy difference was small and not statistically significant. But the efficiency gap was quite significant - ReAct loops narrate a lot and generate quite a bit more (2.6x in our setup) output tokens (and by extension wall-clock time, about 1.5x longer), even with thinking off."r/AI_Agents

Hybrid Approaches and Challenges
The Problem of Termination: While graph architectures offer more control, the moment a backward edge is added for retries or revisions, it inherits the termination challenges of agent loops. Explicit brakes, such as max transition counts and cost ceilings, are crucial for managing these cycles. "The moment you add one backward edge to a workflow, for a retry or a revise step, it stops terminating by construction and inherits the loop's problem."r/LangChain

Balancing LLM and Deterministic Functions: An effective graph architecture often involves a mix of LLM-driven nodes for complex reasoning and deterministic functions for straightforward tasks like database lookups. This prevents unnecessary token consumption. "The deterministic function nodes are lifesavers too, no point burning tokens on something that's just a database lookup."r/PromptEngineering

Are you interested in specific tools or frameworks that support graph-based agentic architectures?

Related Media
response_carousel_media_0
response_carousel_media_1
Related images from across Reddit
AI Agent Communities

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

agenticAI
4.3K weekly visitors
Join
Subreddit for AI Agents

WebAfterAI
4.3K weekly visitors
Join
The internet was built for humans. AI agents are rebuilding it for themselves. Track the shift - AI traffic data, autonomous agents, agentic infrastructure, and what it all means for the web you thought you knew.

LangChain
21K weekly visitors
Join
LangChain enables every company to own their intelligence. Control, govern, and compound intelligence with an open agent engineering ecosystem.
PromptEngineering
139K weekly visitors
Join
Prompt engineering is the application of engineering practices to the development of prompts - i.e., inputs into generative models like GPT or Midjourney.

cursor
101K weekly visitors
Join
The best way to code with AI - cursor.com
Generated from these posts:
1
 r/PromptEngineering
Loop engineering to graph engineering, and what it does to the prompt
4
·
4
2
 r/PromptEngineering
From Loops to Graphs: The shift in Agent architecture
51
·
34

+10
View all
Is this helpful?

Helpful

Unhelpful
latest ai driven development methods but with human review gates added
r/programming
r/codereview
r/softwarearchitecture
From r/programming, r/codereview, r/softwarearchitecture +6 more
Human review gates in AI-driven development are becoming increasingly crucial for maintaining quality, ensuring accountability, and managing the unique challenges posed by AI's high output and occasional "hallucinations." Redditors discuss how to effectively implement these gates to balance AI's speed with human oversight.

The Necessity of Human Review
Preventing AI Errors: Human review is considered essential to catch errors, "dumb decisions," and "AI slop" that AI agents might produce, especially as AI-generated code can lead to significant bugs and financial losses if unreviewed. "Human in the loop is the only way to avoid AIs dumb decisions, if you're not testing before sending out your product then wtf are you there for?"r/ClaudeAI
.

Accountability and Responsibility: Even with AI-generated code, humans remain ultimately responsible for the output. Without proper human review, teams risk approving code they don't fully understand, leading to a lack of accountability when issues arise. "AI is not and it will not be responsible for code, people are fully responsible for output."r/softwarearchitecture
.

Managing "AI Comprehension Debt": AI can generate code faster than humans can comprehend it, creating a bottleneck. Human review gates are necessary to manage this "AI Comprehension Debt" and ensure that changes are understood and validated before integration. "The amount of code to understand and validate seems to increase faster than our capacity to actually review it."r/amazonemployees
.

Strategies for Effective Human Review Gates
Phased Review and Smaller Increments: Breaking down large AI-generated features into smaller, reviewable increments is a common strategy. This allows for more frequent, manageable reviews and ensures that the team stays in sync with the development process. "Break branches into smaller increments. Let AI formulate a PLAN.md and review this, before the actual is made."r/ChatGPTCoding
.

Automated Testing and AI-Assisted Review: Integrating automated tests (unit, integration, E2E) and leveraging AI to review its own work or the work of other agents can offload some of the burden from human reviewers. This helps verify code quality and adherence to standards before human intervention. "For us the biggest shift was moving the verification into CI rather than trying to review diffs by hand. We run a subagent with a fresh context and the test suite against every PR, and a protected-paths hook that blocks changes to auth/billing/migrations unless a human explicitly approves."r/ClaudeAI
.

Focusing Human Review on Critical Areas: Humans should prioritize reviewing crucial aspects such as architecture, business requirements, and high-risk changes (e.g., database migrations, security, billing). This ensures that human attention is directed where it adds the most value, rather than on boilerplate code. "The gates that held up for me are the ones where undoing the step is expensive. The per-stage approvals died on their own, I stopped actually reading them after the first couple and just moved things along... Now the plan gets one real read before implementation starts, and the things that hurt to undo (schema changes, anything pointed at prod data) still need a separate yes."r/ClaudeAI
.

Challenges of Human Review in AI-Driven Development
Reviewer Fatigue and Overload: The high volume of AI-generated code can overwhelm human reviewers, leading to "rubber-stamping" of code without meaningful review. This issue is exacerbated by massive pull requests that are difficult to digest. "Ain't nobody got time for reviewing 10k+ lines of ai slop."r/ChatGPTCoding
.

Maintaining Context and Quality: Ensuring that human reviewers have sufficient context to understand complex AI-generated changes, especially in established codebases, is challenging. AI can also produce code that, while functional, violates existing architectural patterns. "An agent that goes 8 loops deep on its own has already committed to an architecture you never saw form."r/ChatGPTCoding
.

Bias in Review: Reviewers may exhibit bias, being more suspicious of AI-generated code and less rigorous with code from trusted human teammates, regardless of actual code quality. This can lead to inefficient review processes. "Over-scrutinizing AI-written code means reviewers spend disproportionate time on stuff that's actually fine, while the same reviewers' attention runs out before they get to the PR where it mattered."r/codereview
.

Do you want to know more about tools that help manage human review gates in AI-driven development?

AI Development & Code Communities

ChatGPTCoding
77K weekly visitors
Join
r/ChatGPTCoding is a community for people building, learning, and experimenting with AI-assisted coding. Discuss workflows, questions, technical lessons, news, and projects with any AI model or tool—from ChatGPT and Codex to Claude, Cursor, open-source models, and whatever comes next.

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

programming
203K weekly visitors
Joined
Computer Programming

softwarearchitecture
60K weekly visitors
Join
Dive into discussions on designing, structuring, and optimizing software systems. Share insights on architectural patterns, best practices, and real-world experiences.

ClaudeAI
1.7M weekly visitors
Joined
This is a Claude and Claude Code discussion subreddit to help you make a fully informed decision about using Claude and Claude Code to best effect for your own purposes. ¹⌉ Anthropic does not control or operate this subreddit or endorse views expressed here. ²⌉ If your problem requires Anthropic's help, visit https://support.anthropic.com/ This subreddit is not the right place to fix your account issues. ³⌉ For more help, check the resources below. ⁴⌉ Please read the rules before posting.

codereview
2.6K weekly visitors
Join
A subreddit for the posting and community review of code.
Generated from these posts:
1
 r/ClaudeAI
Has anyone tried Spec-Driven Development with AI + human approval gates?
1
·
29
2
 r/amazonemployees
SDEs: how do you review AI-generated CRs?
47
·
29

+10
View all
Is this helpful?

Helpful

Unhelpful
Agentic SDD method vs latest trends
From r/SoftwareEngineering, r/ADHD_Programmers, r/amazonemployees +7 more
While Agentic SDD (Spec-Driven Development) focuses on AI agents driving the development process from specifications, recent trends highlight a shift towards fine-tuning prompts, leveraging graph-based agent architectures, and implementing robust human review gates to manage the complexity and ensure quality in AI-driven development. Redditors emphasize that balancing AI's speed with human oversight and strategic architectural choices is key to effective implementation.

Prompt Engineering and Context Management
Detailed Planning and Intent Capture: A significant trend is to invest heavily in the planning phase, working with high-effort AI models to analyze and plan before any code generation. This involves thorough back-and-forth questioning to fully understand the problem and solution, challenging AI assertions, and developing detailed implementation plans that include validation steps. "What people seem to miss is that you need to spend 80% of your time analyzing and planning with a high-effort model before you ever let it touch a project file."r/ChatGPTCoding
 .

Context Optimization: Developers are moving away from simply feeding raw message history to agents, which can lead to "quadratic context bloat." Instead, the trend is to replace raw transcripts with immutable, versioned artifact pointers and pass only small manifests of committed decisions downstream. This reduces token usage and improves model focus. "Replace raw message history with immutable, versioned artifact pointers. The agent writes outputs to an isolated object or workspace, passing downstream only a tiny manifest (schema version, validation stamp, and committed decisions)."r/LangChain
 .

Architectural Shifts to Graph-Based Agents
Specialized Multi-Agent Systems: The latest trend in Agentic SDD is moving beyond single, monolithic agents to orchestrate multiple specialized agents in graph-based workflows. Each agent (node in the graph) has a single responsibility, such as research, testing, or reporting, allowing for better output quality, smaller context windows, and easier debugging. "Instead of keeping my notes in a document, I converted everything into a visual infographic... The goal is to build a team of specialized agents that each do one thing well."r/vibecoding
 .

Deterministic Control Flow: Graph engineering shifts the control flow into code, enabling explicit branching and state management. This makes the execution path more reproducible and easier to debug compared to the non-deterministic nature of single-agent loops. "The branching becomes a graph of nodes and edges, closer to a state machine than a block of prose. Each node gets its own small prompt with one job." .

Efficiency in Token Usage: Graph workflows can be significantly more efficient in token usage and wall-clock time. By using specialized nodes and optimized context management, developers can reduce the number of tokens consumed, especially for complex tasks. "Accuracy difference was small and not statistically significant. But the efficiency gap was quite significant - ReAct loops narrate a lot and generate quite a bit more (2.6x in our setup) output tokens..."r/AI_Agents
 .

Human Review and Oversight
Strategic Human-in-the-Loop: While AI accelerates development, human review remains critical, evolving from line-by-line code review to higher-level architectural and business intent validation. This involves human approval at key stages and focusing on "expensive to undo" steps like schema changes or critical data interactions. "Humans approve at gates, and a hook can block an action until a named person signs off."r/ClaudeAI
 .

Automated Verification and Testing: To manage the volume of AI-generated code, automated verification through CI/CD pipelines, subagents running test suites, and protected-path hooks are becoming standard. This offloads mundane checks, allowing humans to focus on higher-level architectural and design consistency. "For us the biggest shift was moving the verification into CI rather than trying to review diffs by hand. We run a subagent with a fresh context and the test suite against every PR, and a protected-paths hook that blocks changes to auth/billing/migrations unless a human explicitly approves."r/ClaudeAI
 .

Are you interested in how these trends impact the speed of development or the cost implications?

AI Development Communities

SpecDrivenDevelopment
2.9K weekly visitors
Join
Spec-driven development is the evolution beyond vibe coding. Instead creating code from single prompt, you begin with a clear specification. This specification serves as a contract and single source of truth, guiding tools and AI agents to generate, test, and validate code.

ChatGPTCoding
77K weekly visitors
Join
r/ChatGPTCoding is a community for people building, learning, and experimenting with AI-assisted coding. Discuss workflows, questions, technical lessons, news, and projects with any AI model or tool—from ChatGPT and Codex to Claude, Cursor, open-source models, and whatever comes next.

LangChain
21K weekly visitors
Join
LangChain enables every company to own their intelligence. Control, govern, and compound intelligence with an open agent engineering ecosystem.

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

ClaudeAI
1.7M weekly visitors
Joined
This is a Claude and Claude Code discussion subreddit to help you make a fully informed decision about using Claude and Claude Code to best effect for your own purposes. ¹⌉ Anthropic does not control or operate this subreddit or endorse views expressed here. ²⌉ If your problem requires Anthropic's help, visit https://support.anthropic.com/ This subreddit is not the right place to fix your account issues. ³⌉ For more help, check the resources below. ⁴⌉ Please read the rules before posting.

vibecoding
349K weekly visitors
Joined
fully give in to the vibes. forget that the code even exists.
Generated from these posts:
1
 r/ChatGPTCoding
Hot take: the agentic workflow is deeply wrong
83
·
164
2
 r/ADHD_Programmers
How has agentic development changed your days?
0
·
42

+10
View all
Is this helpful?

Helpful

Unhelpful
Agentic SDD method vs latest trends and its human reviews burden
r/gatech
r/ChatGPTCoding
From r/gatech, r/ADHD_Programmers, r/ChatGPTCoding +5 more
The latest trends in Agentic Spec-Driven Development (SDD) are shifting how human review is managed, primarily by moving away from line-by-line code review towards higher-level artifact review and leveraging automated verification to reduce the human burden. While AI accelerates development, human oversight remains critical, focusing on architectural intent and high-risk changes rather than every line of AI-generated code.

Shifting the Focus of Human Review
Reviewing Intent and Design: Instead of scrutinizing every line of code, human reviewers are increasingly focusing on the Intent.md, spec.md, and plan.md artifacts generated by AI. This ensures that the overall goals, design, and high-level plans align with business requirements before significant code generation occurs, reducing the cost of correcting errors downstream. "Now the plan gets one real read before implementation starts, and the things that hurt to undo (schema changes, anything pointed at prod data) still need a separate yes."r/ClaudeAI

Gatekeeping High-Impact Actions: Human approval is concentrated at critical gates, particularly for actions that are expensive to undo, such as database schema changes, authentication mechanisms, billing systems, or anything interacting with production data. This ensures that human attention is applied where it mitigates the highest risks. "The gates that held up for me are the ones where undoing the step is expensive."r/ClaudeAI

Human-Readable Views of Machine-Consumable Specs: Specs are increasingly designed to be primarily machine-consumable for agents, but with the ability to generate human-readable views tailored to different roles (e.g., product, QA, developer, architect). This allows humans to understand the relevant aspects without manually traversing complex, interconnected specifications. "Instead of reading four specs to understand one flow, I could just ask: > Explain the current Sales Update flow."r/SpecDrivenDevelopment

Mitigating Human Review Burden
Automated Verification in CI/CD: The most significant trend replacing line-by-line human review is the integration of automated verification into Continuous Integration/Continuous Delivery (CI/CD) pipelines. This includes running subagents with fresh contexts and test suites against every Pull Request (PR), and using protected-path hooks to block changes to critical areas without explicit human approval. "For us the biggest shift was moving the verification into CI rather than trying to review diffs by hand... The combination catches more than I ever did reading line by line."r/ClaudeAI

Independent AI Reviewers: Some workflows incorporate independent AI reviewers as gates, where a separate AI model evaluates the output of the primary agent. This "second pair of eyes" aims to catch issues that the generating agent might miss, especially when the reviewer starts from raw evidence rather than relying on the first agent's framing. "Independent AI review as a gate makes sense once a bad output actually costs something, like a merge or release."r/AI_Agents

Concise and Focused Outputs: To combat "AI fatigue" and overwhelming context, developers are training agents to produce concise outputs and summaries. This includes using planning modes, asking for succinct explanations, and configuring agents for concise output styles, allowing humans to quickly grasp the essence of changes. "If you're using claude code turn on concise output by typing /config then Output style > concise."r/ADHD_Programmers

Remaining Challenges in Human Review
Architectural Drift: Even with automated checks, higher-level architectural drift can still slip through. AI agents may introduce new ways of doing things that are functional in isolation but violate established patterns, requiring human reviewers to maintain architectural consistency. "The part that still slips through is higher-level architectural drift. An agent will happily add a fourth way to do something when you already have three."r/ClaudeAI

Reviewer Fatigue and Rubber-Stamping: The sheer volume of AI-generated content can still lead to human reviewers becoming fatigued and "rubber-stamping" approvals without thorough review. This is particularly problematic if human gates are not enforced, leading to a quick "devolve into 'humans never look at it.'"r/ClaudeAI

Are you interested in how these review mechanisms impact the overall development velocity or the cost of AI-driven development?

AI Development & Agent Communities

SpecDrivenDevelopment
2.9K weekly visitors
Join
Spec-driven development is the evolution beyond vibe coding. Instead creating code from single prompt, you begin with a clear specification. This specification serves as a contract and single source of truth, guiding tools and AI agents to generate, test, and validate code.

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

aiagents
21K weekly visitors
Join
AI Agents may be one of the primary conceptual frameworks for humanity’s deployment of applied AI. Tools, commentary and talent

ChatGPTCoding
77K weekly visitors
Join
r/ChatGPTCoding is a community for people building, learning, and experimenting with AI-assisted coding. Discuss workflows, questions, technical lessons, news, and projects with any AI model or tool—from ChatGPT and Codex to Claude, Cursor, open-source models, and whatever comes next.
ADHD_Programmers
25K weekly visitors
Join
Users are important! Without users, reddit would be little more than chunks of code on a server. We love reddit! We hope to make it a better place for everyone!

LocalLLaMA
877K weekly visitors
Join
Subreddit to discuss locally hostable AI. Hi Angelos!
Generated from these posts:
1
 r/ClaudeAI
Has anyone tried Spec-Driven Development with AI + human approval gates?
1
·
29
2
 r/ClaudeAI
Anthropic published an AI-native SDLC playbook. The interesting part isn't the six stages, it's what replaces line-by-line review
114
·
55
r/ClaudeAI - Anthropic published an AI-native SDLC playbook. The interesting part isn't the six stages, it's what replaces line-by-line review

+10
View all
Is this helpful?

Helpful

Unhelpful
Agentic SDD method vs latest trends and its human reviews burden, speed vs quality
r/singularity
r/softwarearchitecture
From r/cscareerquestionsuk, r/singularity, r/softwarearchitecture +8 more
The latest trends in Agentic Spec-Driven Development (SDD) are grappling with the fundamental trade-off between speed and quality, particularly concerning the burden of human review. While AI agents promise accelerated development, maintaining quality and managing human oversight is proving to be a complex challenge.

The Speed vs. Quality Trade-off
AI Speed, Human Bottleneck: AI agents can generate code at an unprecedented pace, potentially achieving 10x speed increases. However, this speed often creates a bottleneck at the human review stage, as teams struggle to keep up with the volume of AI-generated code while ensuring its quality and preventing the accumulation of technical debt. "My position is that we should ship at whatever speed the agents can produce, since that is the entire point of paying for them."r/ChatGPTCoding

The "Code Nobody Can Explain" Problem: Shipping code as fast as agents can produce it risks creating a codebase that no human on the team fully understands, leading to significant "AI comprehension debt" and making debugging or future modifications incredibly difficult. This is seen as a major quality concern, even if the code technically "works." "Your CTO is right about the failure mode but wrong about where the fix goes. Review isn't the bottleneck, missing intent is. When an agent writes 600 lines and nobody wrote down what those lines were supposed to do, review turns into archaeology and you skim."r/ChatGPTCoding

Balancing Model Capabilities for Efficiency: The choice of AI model also impacts this trade-off. While smarter models might produce higher quality output, they can be slower. Developers are experimenting with using different models for different phases of agentic SDD, balancing speed and reasoning capabilities to optimize the overall workflow. "Would it therefore be more beneficial to use a smart, capable model that can run at, say, maybe 150 tokens/sec, rather than a theoretically smarter model that only runs at ~10–20 tokens/sec?"r/hermesagent

Impact on Human Review Burden
Shift from Code Review to Intent Review: To manage the increased volume, human review is shifting away from line-by-line code inspection towards reviewing higher-level artifacts like plan.md or intent.md. This allows humans to approve the "what" and "why" before the "how" is fully implemented by the AI, catching fundamental errors earlier and more cheaply. "What actually cut my review time: make the agent produce the plan first, in a file, before code. What's changing, what's out of scope, what should break if it's wrong."r/ChatGPTCoding

Automated Gates Reduce Mundane Review: Tools like "bugbot" or "coderabbit" are being employed to automate the first pass of code review, handling mechanical checks and escalating only what requires human attention. This reduces the burden on human reviewers, allowing them to focus on more complex issues like architectural consistency or business logic. "Bugbot / coderabbit gates every PR for a first pass, and humans only deep-read what it escalates plus anything touching money or auth."r/ChatGPTCoding

Defining "Quality" as Explainability: Some suggest that for AI-generated code, quality should be tied to explainability. This means requiring agents to produce not just code, but also a summary of behavioral changes, potential impacts on other parts of the system, and identifying a human owner who can explain the change. This transforms review into a check of clarity and accountability, rather than just correctness. "Explainability is not a property of the code, it is an artifact you either produce or you do not — and you can make it a merge gate at roughly the same cost as reviewing the diff."r/ChatGPTCoding

Do you want to explore specific strategies for maintaining quality in AI-driven development?

AI Development Communities

SpecDrivenDevelopment
2.9K weekly visitors
Join
Spec-driven development is the evolution beyond vibe coding. Instead creating code from single prompt, you begin with a clear specification. This specification serves as a contract and single source of truth, guiding tools and AI agents to generate, test, and validate code.

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

ChatGPTCoding
77K weekly visitors
Join
r/ChatGPTCoding is a community for people building, learning, and experimenting with AI-assisted coding. Discuss workflows, questions, technical lessons, news, and projects with any AI model or tool—from ChatGPT and Codex to Claude, Cursor, open-source models, and whatever comes next.

ClaudeCode
716K weekly visitors
Joined
a community for building, learning, and sharing with claude code.

cursor
101K weekly visitors
Join
The best way to code with AI - cursor.com

ClaudeAI
1.7M weekly visitors
Joined
This is a Claude and Claude Code discussion subreddit to help you make a fully informed decision about using Claude and Claude Code to best effect for your own purposes. ¹⌉ Anthropic does not control or operate this subreddit or endorse views expressed here. ²⌉ If your problem requires Anthropic's help, visit https://support.anthropic.com/ This subreddit is not the right place to fix your account issues. ³⌉ For more help, check the resources below. ⁴⌉ Please read the rules before posting.
Generated from these posts:
1
 r/ChatGPTCoding
How do you manage quality when AI agents write code faster than humans can review it?
13
·
23
2
 r/LocalLLM
Best quant & context tradeoff for Qwen 3.8 27B for agentic coding ?
5
·
20
r/LocalLLM - Best quant & context tradeoff for Qwen 3.8 27B for agentic coding ?

+10
View all
Is this helpful?

Helpful

Unhelpful
Impact of automation on code review processes
From r/agile, r/QualityAssurance, r/SoftwareEngineering +7 more
The automation of code review processes significantly impacts the human review burden, often shifting the nature of human involvement while aiming to improve both speed and quality. Redditors observe that while AI tools can accelerate initial reviews and catch common errors, they also introduce new challenges related to review volume and the need for high-level human oversight.

Reduction in Low-Level Review Burden
Automated First Pass: AI review tools like Coderabbit can perform a first pass on pull requests (PRs), catching many low-level issues, linting errors, and even subtle bugs that humans might miss due to fatigue. This automation saves significant human review time on mundane checks. "We started using an AI review tool about 4 months ago to do a first pass before human review. most of what it catches is linter level stuff honestly, const vs let, function too long, whatever."r/codereview

Catching Critical Bugs: Automated tools have demonstrated the ability to identify critical bugs, such as authentication vulnerabilities, that human reviewers overlooked. This highlights their potential to enhance quality by acting as an additional safety net. "last month it flagged an endpoint where we had auth on the route but nothing checking if the user actually owned the resource. any logged in user could pull another users data by guessing the id. three of us missed it in manual review."r/codereview

Shifting Human Review Focus
Focus on High-Level Concerns: With AI handling lower-level checks, human reviewers can concentrate on architectural integrity, business logic, system-wide implications, and ensuring that changes align with product knowledge—areas where AI currently struggles. "the 4 that actually mattered, a wrong feature flag default, a migration that would have locked a table, two auth edge cases, the bot caught 3 and the human caught the flag default, because he knew the product and the bot did not. So the"r/codereview

Context and Intent Review: The human role increasingly involves reviewing the intent and plan behind AI-generated code, rather than dissecting every line. This means ensuring the AI's output correctly interprets the requirements and fits into the broader system. "If you know how to review code, you can have ai review it sufficiently by reviewing its review."r/ClaudeCode

Challenges and New Burdens
Increased PR Volume and Fatigue: AI's ability to generate code quickly leads to a massive increase in the number and size of PRs, which can overwhelm human reviewers and lead to fatigue or "rubber-stamping" approvals. "Output is definitely up, and the results are genuinely good in most cases, but now I am spending most of my week reviewing code and the amount of PR's is crazy."r/softwareengineer

Maintaining Code Comprehension: The rapid generation of code can result in "AI Comprehension Debt," where the codebase grows faster than humans can understand it. This makes future debugging, maintenance, and further development more challenging, even if the code passes initial reviews. "we’re generating code faster than we can review and understand it."r/QualityAssurance

Lack of Big Picture Understanding in AI: While AI excels at finding violations of documented rules and patterns, it often lacks the intuition and comprehensive understanding of the larger system, product goals, or business context to identify deeper architectural or design flaws. "How good is AI code review? when does it work and where is it weak? Does code review still take a long time? Is it good for repeated code? It's probably good for logic. Is it good for application architecture? This is the part I imagine it is"r/ExperiencedDevs

Do you think AI review tools are becoming indispensable for managing code quality and review burden?

Communities for Code Review Insights

codereview
2.6K weekly visitors
Join
A subreddit for the posting and community review of code.

ChatGPTCoding
77K weekly visitors
Join
r/ChatGPTCoding is a community for people building, learning, and experimenting with AI-assisted coding. Discuss workflows, questions, technical lessons, news, and projects with any AI model or tool—from ChatGPT and Codex to Claude, Cursor, open-source models, and whatever comes next.
QualityAssurance
37K weekly visitors
Join
Anything software QA -related; tools, processes, questions etc. As long as it's relevant you can post or ask whatever you like. Spam is forbidden. Spam masquerading as low-effort articles will be dealt with and link posts have been disabled. If you are submitting an article or video, put a description before the link. We like newbies. There are stickied threads for newbies, no they don't have to use them. Please do not message mods to shut down newbie threads.
softwareengineer
62K weekly visitors
Join
A place for software engineers to discuss the profession, careers, industry trends, and the craft of building software. Whether you're just starting out or have decades of experience, thoughtful discussion and meaningful engagement are encouraged.

ExperiencedDevs
216K weekly visitors
Join
For experienced developers. This community should be specialized subreddit facilitating discussion amongst individuals who have gained some ground in the software engineering world. Any posts or comments that are made by inexperienced individuals (outside of the weekly Ask thread) should be reported. Anything not specifically related to development or career advice that is _specific_ to Experienced Developers belongs elsewhere. Try /r/work, /r/AskHR, /r/careerguidance, or /r/OfficePolitics.

cscareerquestions
603K weekly visitors
Joined
CSCareerQuestions is a community for those who are in the process of entering or are already part of the computer science field. Our goal is to help navigate and share challenges of the industry and strategies to be successful .
Generated from these posts:
1
 r/codereview
how are you actually reviewing AI generated code at this point?
17
·
77
2
 r/softwareengineer
Has code review become the whole job for anyone else?
95
·
58

+10
View all
Is this helpful?

Helpful

Unhelpful
SDD steps after initial plan
r/opencode
r/AI_Agents
r/cursor
From r/opencode, r/AI_Agents, r/cursor +4 more
After an initial plan is established in Agentic Spec-Driven Development (SDD), the workflow typically involves several distinct stages that move from high-level understanding and specification to detailed implementation, testing, and continuous refinement. This multi-phase approach often leverages different AI models for specific tasks to optimize cost and performance.

Phased Execution of the Plan
Exploration and Analysis: Before coding begins, agents are often tasked with thoroughly exploring the existing codebase, analyzing dependencies, and identifying patterns. This phase aims to build a deep understanding of the project's current state and identify potential impacts of the proposed changes. "This is where the agent starts exploring the codebase in depth. It reads existing implementations, follows dependencies, analyzes test suites, and identifies patterns across the repository."r/cursor

Detailed Specification and Design: Following exploration, the agent refines the initial plan into a more detailed specification. This stage focuses on structuring the approach, evaluating alternatives, considering trade-offs, and proposing a concrete direction for implementation before any actual coding. "At this stage the objective is not to write code but to think through possible approaches. The agent evaluates alternatives, considers trade-offs, and proposes a direction before any implementation work begins."r/cursor

Iterative Implementation with Deviation Tracking: The coding agent then proceeds to implement the plan in phases, often tracking its progress against a structured work plan and noting any deviations. This allows for continuous adjustment of the design decisions and plan to reflect the actual implementation. "The coding agent is instructed approach phases in order, to keep checkboxes up to date as it progresses, and to list any deviation in a notes section."r/ClaudeCode

Managing Context and Continuity
Centralized and Layered Context: Maintaining an up-to-date and accessible context for the agents is crucial. This often involves using a primary AGENTS.md file for global rules and potentially nested AGENTS.md files in subdirectories for more specific, localized context, ensuring agents have relevant information without being overloaded. "You could add agents md in subdirectories to progressively expose info based on the folder your within."r/cursor

Session Continuity and Memory Management: To prevent agents from "drifting" or losing context across sessions, mechanisms for persistent memory are employed. This can range from simple markdown files that agents are instructed to read and update, to more sophisticated layered indexing or specialized memory plugins. "If context isn’t locked early the agent drifts and reviews get weird fast."r/AI_Agents

Regular Context Refresh: To combat context bloat and ensure accuracy, some Redditors advocate for regularly regenerating or summarizing context files. This prevents the agent from becoming confused by large or contradictory historical information. "weeks, does it stay useful as the project grows, or does it get unwieldy? Too many file stuff going stale, the AI reading the wrong one ,"r/ClaudeCode

Post-Implementation Steps
Testing and Verification: After code generation, rigorous testing and verification are essential. This often includes running comprehensive test suites and potentially using separate sub-agents specifically for verification to ensure the implemented code meets the specified requirements and functions correctly. "The coding agent is instructed approach phases in order, to keep checkboxes up to date as it progresses, and to list any deviation in a notes section. Once the task is complete, we iterate through the deviation notes and adjust the design decisions and plan to incorporate the deviations so by the time they are merged they precisely match what was done."r/ClaudeCode

Documentation and ADR Updates: The final stage involves updating all relevant documentation, including Architectural Decision Records (ADRs) and specifications, to accurately reflect the implemented changes and any deviations. This ensures that the "source of truth" remains consistent and accessible for future reference. "If tweaks to an approach are required, a new ADR is created."r/ClaudeCode

Is your team currently struggling with maintaining context or ensuring consistency in your agentic development workflows?

AI Coding & Agent Communities

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

cursor
101K weekly visitors
Join
The best way to code with AI - cursor.com

opencode
121K weekly visitors
Join
Welcome to the OpenCode subreddit. OpenCode is a popular, free, open-source AI coding agent that integrates powerful AI models directly into your terminal, IDE, or desktop, allowing developers to write, refactor, debug, and explain code using natural language commands. OpenCode supports OpenAI, Anthropic, Gemini models as well as fully local models. Find OpenCode here https://opencode.ai/

opencodeCLI
107K weekly visitors
Join
r/opencodeCLI is a community-driven subreddit for sharing resources, discussions, and tips around OpenCode which is a Go + TypeScript open-source CLI TUI for coding assistance. It supports multiple providers (Anthropic Claude, OpenAI, Gemini, local models, etc.)

PiCodingAgent
51K weekly visitors
Join
An unofficial community dedicated to the minimalist Pi coding agent. Adapt pi to your workflows, not the other way around, without having to fork and modify pi internals. Join the PI revolution now!

ClaudeCode
716K weekly visitors
Joined
a community for building, learning, and sharing with claude code.
Generated from these posts:
1
 r/cursor
I use a 9-agent SDD harness where each phase uses a different model. The total cost is $10-15/month. Here's the full breakdown.
36
·
61
2
 r/ClaudeCode
Spec-Driven Development (SDD) frameworks vs. AI "Plan Mode" for large-scale projects?
14
·
36

+10
View all
Is this helpful?

Helpful

Unhelpful
Best practices for integrating human reviews
r/PinoyProgrammer
From r/softwaredevelopment, r/PinoyProgrammer, r/InternalAudit +7 more
Integrating human reviews into Agentic Spec-Driven Development (SDD) is crucial for maintaining quality and managing risks. The core challenge lies in balancing the speed of AI with meaningful human oversight, preventing "rubber-stamping," and ensuring human reviewers have sufficient context without being overwhelmed.

Establishing Clear Human Approval Gates
Approval Gates for High-Impact Actions: Implement mandatory human approval gates for critical actions such as deploying to production, emailing customers, or making irreversible changes like schema modifications. This ensures human judgment is applied where the "blast radius" is highest. "The gates that held up for me are the ones where undoing the step is expensive."r/ClaudeAI

Focus on Intent, Not Just Code: Shift the review focus from line-by-line code inspection to higher-level artifacts like plan.md or intent.md. Humans should approve the "what" and "why" before the AI generates the "how," allowing for early correction of fundamental design flaws. "Review will move from 'read every line' to 'check whether intent, spec, tests, and incident notes still agree.'"r/ClaudeAI

Optimizing the Review Process
Small, Digestible Changes: Instruct AI agents to make small, easily reviewable changes. This practice, already a best practice in traditional development, becomes even more critical with AI-generated code to prevent reviewer fatigue and improve comprehension. "I ask my team to tell AI to make small changes so it’s easier to digest and review."r/amazonemployees

Adversarial Review by AI: Utilize a separate "adversarial reviewer" AI agent to scrutinize the code generated by the primary agent. This can catch problems before human review, and some Redditors even run multiple rounds of adversarial review for complex tasks. "I have another Fable agent perform an adversarial review on the whole thing...It always finds problems."r/ClaudeAI

Context-Rich Summaries for Reviewers: When an agent requests human approval, it should provide a concise, plain-language explanation of what it plans to do, why, and the potential consequences. This helps reviewers make informed decisions quickly without sifting through excessive detail. "My orchestrator has a similar rule where is has to not only explain the issue and propose a recommendation, both in plain language, but it also has to provide the strongest argument against its own recommendation."r/agenticAI

Addressing Challenges in Human Oversight
Preventing "Rubber-Stamping": Actively combat the tendency for humans to mindlessly approve AI output due to volume or fatigue. This can involve making reviews more engaging, ensuring the impact of approval is clear, and training reviewers on automation bias. "The rubber-stamp problem. The whole point of a human gate is judgment. But the moment volume goes up, the human stops reading and just clicks approve."r/agenticAI

Managing Asynchronous Approvals: Develop robust mechanisms for agents to handle waiting for human approval, especially when humans are offline. Options include queuing tasks, holding state, or having agents work on other tasks that don't require immediate human intervention. "The agent is waiting and the human is asleep... Every option has a failure mode."r/agenticAI

Are you interested in specific tools or techniques for making AI-generated explanations more concise for human review?

AI Agent Communities

AI_Agents
275K weekly visitors
Join
A place for discussion around the use of AI Agents and related tools. AI Agents are LLMs that have the ability to "use tools" or "execute functions" in an autonomous or semi-autonomous (also known as human-in-the-loop) fashion. Follow our event calendar: https://lu.ma/oss4ai Join us on Discord! https://discord.gg/6tGkQcFjBY

AIAssisted
28K weekly visitors
Join
r/AIAssisted is for people who actually use AI, not just talk about it. We swap prompts, expose broken tools, share wins (and fails), and post the stuff no one's putting in tutorials. If you're tired of the hype and want real insight, welcome.

LocalLLaMA
877K weekly visitors
Join
Subreddit to discuss locally hostable AI. Hi Angelos!

agenticAI
4.3K weekly visitors
Join
Subreddit for AI Agents

ClaudeAI
1.7M weekly visitors
Joined
This is a Claude and Claude Code discussion subreddit to help you make a fully informed decision about using Claude and Claude Code to best effect for your own purposes. ¹⌉ Anthropic does not control or operate this subreddit or endorse views expressed here. ²⌉ If your problem requires Anthropic's help, visit https://support.anthropic.com/ This subreddit is not the right place to fix your account issues. ³⌉ For more help, check the resources below. ⁴⌉ Please read the rules before posting.
softwaredevelopment
12K weekly visitors
Join
Software development methodologies, techniques, and tools. Not to be confused with programming. Covering Agile, RUP, Waterfall, Crystal, Extreme Programming, Scrum, Lean, Kanban and more.
Generated from these posts:
1
 r/ClaudeAI
Has anyone tried Spec-Driven Development with AI + human approval gates?
1
·
29
2
 r/ClaudeAI
Anthropic published an AI-native SDLC playbook. The interesting part isn't the six stages, it's what replaces line-by-line review
114
·
55
r/ClaudeAI - Anthropic published an AI-native SDLC playbook. The interesting part isn't the six stages, it's what replaces line-by-line review

+10
View all
Is this helpful?

Helpful

Unhelpful
Related
best tools for AI code review
strategies for effective human oversight
challenges in AI-assisted development
Responses are AI-generated from posts and comments and may not be accurate.
