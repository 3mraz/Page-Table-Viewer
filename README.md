# Cysec Project: Page-Table Viewer

**Student**: Amr Jaafar

**Advisor**: Fabian Thomas

**Project Goals**: Microarchitectural attacks often require interaction with low-level data structures, such as page tables. On Linux there is already a tool (PTEditor) that can read and write Linux page tables from C/C++ code, but there is no interactive viewer/editor yet. This project is about engineering such a viewer/editor based on PTEditor. The tool should provide a GUI that visualizes the page table and allows for live patching. Further, it should run as a statically linked Web App on remote machines, so that easy access via SSH port forwarding is possible.

**Handout**: 13.05.2024

## Structure

Initially, this repository contains the following:

- `README.md`: This readme. **Read the entire readme!**
- `requirements.md`: Should be updated by your supervisor once project goals and requirements are clear.

You can remove/change/add anything you want, the initial structure doesn't have to be kept.

## Documentation

Provide a brief description of what can be found in this repository.

If your implementation is not in your repository, provide a link (or similar) to where it can be found.

If you include your code, clearly describe how to set up, compile, run, and use your code.
Also describe known issues and potential future work.

## Contact

You should have been assigned an advisor, who is your direct point of contact.
If this did not happen, ask Michael Schwarz.
Likely, there is also a Mattermost channel about your project.
For any questions, problems, and progress reports, post in the Mattermost channel, as this ensures that everyone involved in the project is kept up to date and can help your.
For personal questions, or if you want to schedule a meeting, directly contact your advisor.
If there is a problem with your advisor (e.g., you don't get an answer in time), contact Michael Schwarz.

## Checklist

- [x] Get a project and an advisor.
- [ ] Discuss project goals and requirements with your advisor (Requirements should be documented in `requirements.md`).
- [ ] Have fun with the project (Remember to start with small steps).
- [ ] Submit everything (documentation, code) before the deadline in this repository.
- [ ] Wait for the grade.

## FAQ

- **I asked a question in the Mattermost/via mail but did not receive an answer, what should I do?**

Generally, we try to answer as quickly as possible. However, sometimes, it can take up to a week for us to answer. If you did not get an answer within a week, poke us (e.g., by writing another mail/message).

- **Is the submission deadline of the project a hard deadline?**

Yes.

- **I need software/hardware/guidance/.... Whom should I contact?**

Ask in the Mattermost channel. In the unlikely case that there is no answer, contact your advisor directly.

- **I worked on the topic, but I'd rather change the topic, can I do that?**

No. But you are free to stop working on the project and look for some other project in the next semester.
However, please tell us if you do that, and don't simply ghost us.
