<script>
	import Modal from './Modal.svelte';
	import { enhance } from '$app/forms';
	import { fly, fade } from 'svelte/transition';

	let creatingTodo = false;
  let showModal =false;

	export let data;
	export let form;
</script>

<div>
	<form
		method="POST"
		action="?/createTodo"
		use:enhance={() => {
			creatingTodo = true;
			return async ({ update }) => {
				await update();
				creatingTodo = false;
			};
		}}
	>
		<input aria-busy={creatingTodo} type="text" name="content" placeholder="Create a new todo..." />
		{#if creatingTodo}
			<small><small class="spinner" />Posting your todo to the database..</small>
		{/if}

		{#if form?.success === false && !creatingTodo}
			<p class="error">Error: {form?.message}</p>
		{/if}
	</form>

	<!--
	<pre>{JSON.stringify(data.user, null, 2)}</pre>
	<pre>{JSON.stringify(data.todos, null, 2)}</pre>
  -->

	<div class="todos">
		{#each data.todos as todo (todo.id)}
			<Modal {todo} {showModal} />

			<div in:fly={{ y: -120, duration: 120 }} out:fade={{ duration: 200 }} class="todo">
				<input type="checkbox" value={todo.done} name="done" />

				<span style={todo.done ? 'text-decoration: line-through' : ''}>{todo.content}</span>
				<div class="todoControls">
					<button
						class="iconbutton"
						data-target="modal-example"
						on:click={() => {
							showModal = !showModal;
						}}
					>
						<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24"
							><path
								fill="currentColor"
								d="M14 11c0 .55-.45 1-1 1H4c-.55 0-1-.45-1-1s.45-1 1-1h9c.55 0 1 .45 1 1zM3 7c0 .55.45 1 1 1h9c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1zm7 8c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1s.45 1 1 1h5c.55 0 1-.45 1-1zm8.01-2.13l.71-.71a.996.996 0 0 1 1.41 0l.71.71c.39.39.39 1.02 0 1.41l-.71.71l-2.12-2.12zm-.71.71l-5.16 5.16c-.09.09-.14.21-.14.35v1.41c0 .28.22.5.5.5h1.41c.13 0 .26-.05.35-.15l5.16-5.16l-2.12-2.11z"
							/></svg
						>
					</button>

					<form method="POST" action="?/deleteTodo" use:enhance>
						<input type="hidden" value={todo.id} name="id" />
						<button class="iconbutton">
							<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
								<path
									fill="var(--del-color)"
									stroke="none"
									d="M22 4.2h-5.6L15 1.6c-.1-.2-.4-.4-.7-.4H9.6c-.2 0-.5.2-.6.4L7.6 4.2H2c-.4 0-.8.4-.8.8s.4.8.8.8h1.8V22c0 .4.3.8.8.8h15c.4 0 .8-.3.8-.8V5.8H22c.4 0 .8-.3.8-.8s-.4-.8-.8-.8zM10.8 16.5c0 .4-.3.8-.8.8s-.8-.3-.8-.8V10c0-.4.3-.8.8-.8s.8.3.8.8v6.5zm4 0c0 .4-.3.8-.8.8s-.8-.3-.8-.8V10c0-.4.3-.8.8-.8s.8.3.8.8v6.5z"
								/>
							</svg>
						</button>
					</form>
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	div {
		margin-bottom: 15px;
	}

	.todo {
		display: table;
		border-radius: var(--border-radius);
		background: var(--code-background-color);
		padding: 10px;
	}

	.todo > span {
		width: 100%;
		display: table-cell;
		vertical-align: middle;
		text-align: center;
	}
	.todoControls {
		display: inline-flex;
		flex-direction: column;
		margin: auto;
	}
	.iconbutton {
		margin: 0;
		padding: 0;
		border: 0;
		padding-top: 10px;
		line-height: inherit;
		background-color: inherit;
		opacity: 0.5;
	}
	.iconbutton:hover {
		opacity: 0.9;
	}

	.error {
		color: var(--del-color);
		font-weight: bold;
		text-align: center;
	}
	.spinner {
		display: inline-block;
		width: 1em;
		height: 1em;
		border: 0.1875em solid currentColor;
		border-right-color: currentcolor;
		border-radius: 1em;
		border-right-color: transparent;
		content: '';
		vertical-align: text-bottom;
		vertical-align: -0.125em;
		animation: spinner 0.75s linear infinite;
		opacity: var(--loading-spinner-opacity);
		margin-left: 1em;
		margin-right: 1em;
	}
</style>
