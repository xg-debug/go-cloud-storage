<template>
    <div id="app">
        <router-view />
    </div>
</template>

<style>
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

html, body {
    height: 100%;
    overflow: hidden;
}

#app {
    height: 100vh;
    background: var(--cb-bg);
    color: var(--cb-text);
}
</style>
